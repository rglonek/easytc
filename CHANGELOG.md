# CHANGELOG

## v0.2

* Add verbose logging option
* Fix bug where if a destination port was specified, and source port wasn't, an infinite loop would prevent proper function
* Make interface name optional in `set` and `del` - if not specified, this will apply to all interfaces except `lo`
* Test for `sch_netem` kernel module and error accordingly
* Fix `priomap` to use a single `TOS`
* Start rules from band `1:4` since `1:3` is default (should be no filter)
* Add fallback for `tc -j filter/qdisc show ...` - support text parsing for broken or unsupported json output in older distros
* Test support for ubuntu: `24.04, 22.04, 20.04`, rocky/centos: `9, 8, 7`, debian `13, 12, 11, 10`

## v0.1

* Initial Release
