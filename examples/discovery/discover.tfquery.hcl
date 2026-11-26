list "pfsense_firewall_alias" "aliases" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_firewall_nat_npt" "npt" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_firewall_nat_one_to_one" "one_to_one" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_firewall_nat_port_forward" "port_forwards" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_firewall_rule" "rules" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_firewall_schedule" "schedules" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_firewall_altq" "altq" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_firewall_limiter" "limiters" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_firewall_pflow_exporter" "exporters" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_firewall_virtual_ip" "virtual_ips" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_wireguard_settings" "settings" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_wireguard_tunnel" "tunnels" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_wireguard_peer" "peers" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_haproxy_resolver" "resolvers" {
  provider = pfsense-nexus
  limit    = 10000
}

list "pfsense_mdns" "mdns" {
  provider = pfsense-nexus
  limit    = 10000
}
