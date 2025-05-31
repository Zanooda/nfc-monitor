SUMMARY = "NFC Monitor Tool"
DESCRIPTION = "Tool to monitor NFC readers and report tag arrival/departure"
LICENSE = "CLOSED"

SRC_URI = "git://github.com/Zanooda/nfc-monitor.git;protocol=https;branch=main"
SRCREV = "${AUTOREV}"

S = "${WORKDIR}/git"

DEPENDS = "libnfc"
RDEPENDS:${PN} = "libnfc"

inherit go-mod

GO_IMPORT = "nfc-monitor"

do_compile() {
    export GOOS="linux"
    export GOARCH="arm"
    export GOARM="7"
    export CGO_ENABLED="1"
    export CC="${CC}"
    export CXX="${CXX}"
    export CGO_CFLAGS="${CFLAGS}"
    export CGO_LDFLAGS="${LDFLAGS}"
    export GOPATH="${WORKDIR}/go"
    export GOCACHE="${WORKDIR}/go/.cache"
    
    cd ${S}
    ${GO} mod download
    ${GO} build -ldflags="-s -w" -o ${B}/nfc-monitor .
}

do_install() {
    install -d ${D}${bindir}
    install -m 0755 ${B}/nfc-monitor ${D}${bindir}/
}

FILES:${PN} = "${bindir}/nfc-monitor"
