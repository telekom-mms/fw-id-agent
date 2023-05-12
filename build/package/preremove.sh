#!/bin/sh -e
# taken from https://github.com/Debian/debhelper/blob/master/dh

UNIT='fw-id-agent.service'

case "$1" in
  'remove')
    if [ -z "${DPKG_ROOT:-}" ] && [ -d /run/systemd/system ] ; then
      deb-systemd-invoke --user stop $UNIT >/dev/null || true
    fi
    ;;
esac