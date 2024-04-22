# Firewall Identity Agent

Firewall Identity Agent is a systemd user service that logs the current user in
on a Firewall Identity Service. It uses [Trusted Network
Detection](https://github.com/telekom-mms/tnd/) to detect if the host is
currently connected to a trusted network and then logs the user in on the
Firewall Identity Service. Desktop notifications inform the user about the
trusted network and login state.

## Installation

For installation you can chose between 2 options:

### Using Debian/Ubuntu package

Download the package from releases page and use the following instructions to install and activate the agent:

```console
$ sudo apt install ./fw-id-agent.deb
$ sudo cp /usr/share/doc/fw-id-agent/examples/config.json /etc/fw-id-agent.json # and adjust config parameters
$ sudo systemctl --user start fw-id-agent.service
```

### Using tar.gz archive

Download the archive from releases page and use the following instructions to install and activate the agent:

```console
$ tar -xf fw-id-agent.tar.gz && cd <extracted directory>
$ sudo cp example_config.json /etc/fw-id-agent.json # and adjust config parameters
$ sudo cp fw-id-agent /usr/bin/
$ sudo cp fw-id-cli /usr/bin/
$ sudo cp fw-id-agent.service /usr/lib/systemd/user/
$ sudo systemctl --user enable fw-id-agent.service
$ sudo systemctl --user start fw-id-agent.service
```

## Usage

There are two executables: `fw-id-agent` is the Firewall Identity Agent and
`fw-id-cli` is the command line interface for the Firewall Identity Agent.

### fw-id-agent

If you want to run the Firewall Identity Agent manually, you can run the
`fw-id-agent` executable with the following command line arguments:

```
Usage of fw-id-agent:
  -config file
        Set config file
  -keepalive minutes
        Set default client keep-alive in minutes (default 5)
  -logintimeout seconds
        Set client login request timeout in seconds (default 15)
  -logouttimeout seconds
        Set client logout request timeout in seconds (default 5)
  -notifications
        Set desktop notifications (default true)
  -realm string
        Set kerberos realm
  -retrytimer seconds
        Set client login retry timer in case of errors in seconds (default 15)
  -serviceurl string
        Set service URL
  -startdelay seconds
        Set agent start delay in seconds
  -tndservers list
        Set comma-separated list of TND server url:hash pairs
  -verbose
        Set verbose output
  -version
        print version
```

For example, you can run the Firewall Identity Agent with the following command
line:

```console
$ fw-id-agent -config /etc/fw-id-agent.json
```

### fw-id-cli

You can show and monitor the current status of the Firewall Identity Agent or
send re-login requests using the `fw-id-cli` executable:

```
Usage:
  fw-id-cli [options] [command]

Options:
  -version
        print version

Commands:
  status
        show agent status
  monitor
        monitor agent status updates
  relogin
        relogin agent
```

The `status` command of `fw-id-cli` supports printing verbose or JSON output
with extra command line arguments:

```
Usage of status:
  -json
        set json output
  -verbose
        set verbose output
```

For example, you can show the verbose status with the following command line:

```console
$ fw-id-cli status -verbose
```
