#!/bin/sh -e
# taken from https://git.launchpad.net/ubuntu/+source/debhelper/tree/autoscripts/postrm-systemd-user?h=applied/13.6ubuntu1

UNIT='fw-id-agent.service'

case "$1" in
  'remove')
    if [ -x "/usr/bin/deb-systemd-helper" ]; then
      deb-systemd-helper --user mask $UNIT >/dev/null || true
    fi
    ;;

  'purge')
    if [ -z "${DPKG_ROOT:-}" ] && [ -x "/usr/bin/deb-systemd-helper" ]; then
      deb-systemd-helper --user purge $UNIT >/dev/null || true
      deb-systemd-helper --user unmask $UNIT >/dev/null || true
    fi
    ;;
esac