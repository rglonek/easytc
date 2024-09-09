# EasyTC

Simple program for easy, basic `tc` rule creation and management. Supported features are:
* filter by IP and port
* apply max rate (link speed), latency, packet loss
* show all qdisc, filters and compiled rules
* reset/remove rules

## Note on kernel

The kernel module `sch_netem` is required. Easytc will automatically attempt to `modprobe` it if need be.

On some distros, extra packages must be installed to add `netem` module. For example on RH-based distros: `yum install kernel-modules-extra iproute-tc`.

## CLI Usage

### Help

```
$ ./easytc --help
Usage:
  easytc [OPTIONS] <command>

Help Options:
  -h, --help  Show this help message

Available commands:
  del    delete a tc rule
  reset  remove all tc rules
  set    create a tc rule
  show   list tc rules or interfaces
```

```
$ ./easytc set --help
Usage:
  easytc [OPTIONS] set [set-OPTIONS]

Help Options:
  -h, --help            Show this help message

[set command options]
      -i, --interface=  specify an interface for the rule
      -s, --src-ip=     optional: filter by source IP
      -d, --dst-ip=     optional: filter by destination IP
      -S, --src-port=   optional: filter by source port
      -D, --dst-port=   optional: filter by destination port
      -l, --latency-ms= optional: specify latency (number) of milliseconds
      -p, --loss-pct=   optional: specify packet loss percentage
      -e, --rate-bytes= optional: specify link speed rate, in bytes
```

### Show Interfaces

```
$ ./easytc show iface
enp0s5
```

### Apply a rule

Note that if the `interface` switch is not provided a rule for every interface will be created.

```
$ ./easytc set -s 10.0.0.0/8 -d 8.8.8.8 -l 100 -p 20
```

### Test

```
$ ping 1.1.1.1
PING 1.1.1.1 (1.1.1.1) 56(84) bytes of data.
64 bytes from 1.1.1.1: icmp_seq=1 ttl=128 time=22.9 ms
64 bytes from 1.1.1.1: icmp_seq=2 ttl=128 time=27.8 ms

$ ping 8.8.8.8
PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=128 time=124 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=128 time=127 ms
```

### Show Rules

```
$ ./easytc show rules
 Iface   SrcIP       DstIP    SrcPort  DstPort  LatencyMs  PacketLossPct  RateBytes  TcFlowID  TcQdiscHandle  TcFilterHandle 
-----------------------------------------------------------------------------------------------------------------------------
 enp0s5  10.0.0.0/8  8.8.8.8                    100        20.00                     1:3       30:            800::800   
```

### Show all rules and interfaces, in json format

```
$ ./easytc show all
[...]
```

### Remove all rules

```
$ ./easytc reset
```

## As golang package

All exported functions are defined in `easytc/tc` package, used in `easytc/cli`. See the simple CLI implementation for exact usage.
