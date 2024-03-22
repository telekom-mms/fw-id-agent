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

If you want to run the Firewall Identity Agent manually, you can run the
executable with the following command line arguments:

```
Usage of fw-id-agent:
  -config file
        Set config file (default "config.json")
  -verbose
        Set verbose output
  -version
        print version
```
