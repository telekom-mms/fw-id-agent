#!/bin/sh -e
# taken from https://github.com/Debian/debhelper/blob/master/dh

UNIT='fw-id-agent.service'

case "$1" in
  'remove')
    if [ -d /run/systemd/system ] ; then
      systemctl --user daemon-reload >/dev/null || true
    fi
    if [ -x "/usr/bin/deb-systemd-helper" ]; then
      deb-systemd-helper --user mask $UNIT >/dev/null || true
    fi
    ;;

  'purge')
    if [ -x "/usr/bin/deb-systemd-helper" ]; then
      deb-systemd-helper --user purge $UNIT >/dev/null || true
      deb-systemd-helper --user unmask $UNIT >/dev/null || true
    fi
    ;;
esac