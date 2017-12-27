## DNS resolver for MySQL backed ISC Kea

Kea supports dynamic DNS updates but requires a supported name server. It is also currently limited to updating one (master) server, which may require traditional DNS setups to have a single point for failure for receiving the update requests.

This is a simple tool that responds to DNS requests to directly query the DHCP lease database and return as DNS responses. An instance may be deployed for each MySQL HA member.

Based on DNS library https://github.com/miekg/dns

Included build is for Alpine Linux and intended for use in Docker:
https://hub.docker.com/r/randomcoww/go-kea-lease-resolver/

### Arguments

    -h MySQL host
    -d Database name
    -w Database password
    -p MySQL port
    -u Database user
    -t Kea lease table name (lease4 by default)
    -listen Listen port for DNS requests (uses 53530 by default)

### Unbound    

Hook up to local Unbound for forward and reverse resolver:

    server:
      local-zone: domain.internal nodefault
      local-zone: 168.192.in-addr.arpa nodefault
      private-domain: domain.internal
      domain-insecure: domain.internal
      ...
    remote-control:
      control-enable: yes
    stub-zone:
      name: domain.internal
      stub-addr: 127.0.0.1@53530
    stub-zone:
      name: 168.192.in-addr.arpa
    stub-addr: 127.0.0.1@53530
    
