#!/usr/bin/env bash
# Teste E2E do contrato entre o gateway legado (PHP) e o motor (Go).
#
# Com a stack no ar (docker compose up), envia propostas no formato legado ao
# gateway e confere: a resposta legada, a linha no MySQL legado e a avaliação
# no MongoDB do motor, ligadas pelo mesmo protocolo. Cada lado testa contra um
# dublê do outro; só este teste garante que os dois concordam.
#
# Uso: bash scripts/e2e.sh   (ou make e2e, que sobe a stack antes)
set -uo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
failures=0
accepted=()

ok() { echo "  ok     $1"; }
fail() { echo "  FALHA  $1"; failures=$((failures + 1)); }

check() { # descrição, esperado, obtido
    if [ "$2" = "$3" ]; then ok "$1"; else fail "$1 (esperado: $2, obtido: $3)"; fi
}

proposal() { # centavos, nome, pep, moeda
    printf '{"produto":{"tipo":"EMPRESTIMO_PESSOAL","pais_origem":"US","moeda_origem":"USD"},"operacao":{"valor_centavos":%s,"moeda":"%s","modalidade":"TRANSFERENCIA_INTERNACIONAL","contraparte":{"nome":"%s","pep":"%s"}}}' "$1" "$4" "$2" "$3"
}

post() { # corpo -> STATUS e BODY
    local out
    out=$(curl -s -w $'\n%{http_code}' -X POST "$GATEWAY_URL/propostas" -H 'Content-Type: application/json' -d "$1")
    BODY=${out%$'\n'*}
    STATUS=${out##*$'\n'}
}

field() { # valor escalar de um campo do BODY (texto ou número)
    printf '%s' "$BODY" | sed -n "s/.*\"$1\":\"\{0,1\}\([^,\"}]*\)\"\{0,1\}.*/\1/p"
}

count_adaptations() { # itens da lista "exigencias" do BODY; 0 se ausente ou vazia
    local list
    list=$(printf '%s' "$BODY" | grep -o '"exigencias":\[[^]]*\]')
    if [ -z "$list" ] || [ "$list" = '"exigencias":[]' ]; then
        echo 0
    else
        echo $(( $(printf '%s' "$list" | grep -o '","' | wc -l) + 1 ))
    fi
}

mysql_count() {
    docker exec legacy-mysql mysql -ulegacy -plegacy123 legacy -N -e "$1" 2>/dev/null
}

mongo_count() {
    docker exec regulatory-mongodb mongosh -u admin -p admin123 --authenticationDatabase admin --quiet regulatory --eval "$1"
}

accepted_case() { # id, centavos, nome, pep, situação esperada, IOF esperado, exigências esperadas
    echo "$1: proposta aceita"
    post "$(proposal "$2" "$3" "$4" USD)"
    check "status" 201 "$STATUS"
    check "situacao" "$5" "$(field situacao)"
    check "iof_centavos" "$6" "$(field iof_centavos)"
    check "exigencias" "$7" "$(count_adaptations)"
    local protocol
    protocol=$(field protocolo)
    if [ -n "$protocol" ]; then accepted+=("$protocol"); else fail "protocolo ausente na resposta"; fi
}

rejected_case() { # id, descrição, corpo, campo legado esperado
    echo "$1: $2"
    post "$3"
    check "status" 422 "$STATUS"
    if printf '%s' "$BODY" | grep -q "\"campo\":\"$4\""; then ok "campo $4 apontado"; else fail "campo $4 nao apontado: $BODY"; fi
}

echo "Gateway: $GATEWAY_URL"
rows_before=$(mysql_count "SELECT COUNT(*) FROM propostas")

accepted_case C01 100000 "John Doe" N PENDENTE_ADAPTACAO 1900 1
accepted_case C05 7500000 "John Doe" N COMUNICAR_COAF 142500 2
accepted_case C10 7500000 "Mara Costa" S COMUNICAR_COAF 142500 3
rejected_case G03 "formato legado invalido (rejeitada pelo gateway)" "$(proposal 100000 "John Doe" talvez USD)" operacao.contraparte.pep
rejected_case G04 "moeda nao suportada (rejeitada pelo motor, traduzida)" "$(proposal 100000 "John Doe" N BRL)" operacao.moeda

echo "Persistencia: aceitas nos dois bancos, rejeitadas em nenhum"
for protocol in "${accepted[@]}"; do
    check "MySQL legado tem $protocol" 1 "$(mysql_count "SELECT COUNT(*) FROM propostas WHERE protocolo='$protocol'")"
    check "MongoDB do motor tem $protocol" 1 "$(mongo_count "db.evaluations.countDocuments({_id:'$protocol'})")"
done
rows_after=$(mysql_count "SELECT COUNT(*) FROM propostas")
check "linhas novas no MySQL (so as 3 aceitas)" 3 "$((rows_after - rows_before))"

if [ "$failures" -gt 0 ]; then
    echo "E2E: $failures falha(s)"
    exit 1
fi
echo "E2E: tudo ok"
