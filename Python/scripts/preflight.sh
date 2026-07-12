#!/bin/bash
set -euo pipefail
# MediaCrawlerPro service health check script
# Usage: bash scripts/preflight.sh [platform]
# Output: JSON with service status (and optional cookie check)

check_service() {
    local url=$1
    local http_code
    http_code=$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 2 "$url" 2>/dev/null || echo "000")
    if [ "$http_code" = "200" ]; then
        echo "true"
    else
        echo "false"
    fi
}

SIGNSRV=$(check_service "http://localhost:8989/signsrv/pong")
COOKIEBRIDGE=$(check_service "http://localhost:8274/ping")
DOWNLOADER=$(check_service "http://localhost:8205/ping")

PLATFORM=${1:-}

if [ -z "$PLATFORM" ]; then
    echo "{\"signsrv\": $SIGNSRV, \"cookiebridge\": $COOKIEBRIDGE, \"downloader\": $DOWNLOADER}"
elif [ "$COOKIEBRIDGE" = "true" ]; then
    COOKIE_CHECK=$(curl -s --connect-timeout 2 "http://localhost:8274/api/cookies/$PLATFORM" 2>/dev/null | python3 -c "
import sys,json
try:
    d=json.load(sys.stdin)
    has=bool(d.get('isok') and d.get('data',{}).get('cookies'))
    src=d.get('data',{}).get('source','') if has else ''
    print(json.dumps({'available': has, 'source': src}))
except:
    print(json.dumps({'available': False, 'source': ''}))
" 2>/dev/null || echo '{"available": false, "source": ""}')
    echo "{\"signsrv\": $SIGNSRV, \"cookiebridge\": $COOKIEBRIDGE, \"downloader\": $DOWNLOADER, \"cookie_check\": {\"platform\": \"$PLATFORM\", \"result\": $COOKIE_CHECK}}"
else
    echo "{\"signsrv\": $SIGNSRV, \"cookiebridge\": $COOKIEBRIDGE, \"downloader\": $DOWNLOADER, \"cookie_check\": {\"platform\": \"$PLATFORM\", \"result\": {\"available\": false, \"source\": \"cookiebridge_offline\"}}}"
fi
