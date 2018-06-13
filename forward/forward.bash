function forward() {
    if [ -z "$1" ]; then
        echo "Usage: forward [port]"
        return
    fi

    export PROXY_HOST=127.0.0.1
    export PROXY_PORT=$1

    adb forward tcp:$PROXY_PORT tcp:$PROXY_PORT
    if [ $? -ne 0 ]; then
        return
    fi

    export all_proxy="${PROXY_HOST}:${PROXY_PORT}"
    export http_proxy=$all_proxy
    export https_proxy=$all_proxy
    export ftp_proxy=$all_proxy
    export rsync_proxy=$all_proxy
    export ALL_PROXY=$all_proxy
    export HTTP_PROXY=$all_proxy
    export HTTPS_PROXY=$all_proxy
    export FTP_PROXY=$all_proxy
    export RSYNC_PROXY=$all_proxy
    alias chromium="chromium --proxy-server=$all_proxy"
    alias gradle="gradle -Dhttp.proxyHost=$PROXY_HOST -Dhttp.proxyPort=$PROXY_PORT -Dhttps.proxyHost=$PROXY_HOST -Dhttps.proxyPort=$PROXY_PORT"
    echo proxy configured on $all_proxy
}
