#!/bin/sh -e

case "$1" in
	'configure')
		systemctl daemon-reload
		systemctl --global enable fw-id-agent.service
		;;
esac
