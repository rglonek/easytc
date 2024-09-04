package tc

import (
	"fmt"
	"os/exec"
)

// remove all rules from a given interface; if interface is not given, removes all rules from all interfaces
func Reset(iface *string) error {
	ifaces := []string{}
	if iface == nil {
		var err error
		ifaces, err = ListIface()
		if err != nil {
			return err
		}
	} else {
		ifaces = append(ifaces, *iface)
	}
	for _, i := range ifaces {
		exec.Command("tc", "qdisc", "del", "dev", i, "root").CombinedOutput()
	}
	return nil
}

func CleanupUnusedQdisc() error {
	rules, err := ListRules()
	if err != nil {
		return err
	}
	for _, rule := range rules.Rules {
		if rule.FilterHandle == nil && rule.Iface != nil && rule.QdiscHandle != nil && rule.FlowID != nil {
			out, err := exec.Command("tc", "qdisc", "del", "dev", *rule.Iface, "parent", *rule.FlowID, "handle", *rule.QdiscHandle).CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s:%s", err, string(out))
			}
		}
	}
	return nil
}

func Delete(r *Rule) error {
	// list qdisc
	rules, err := ListRules()
	if err != nil {
		return err
	}

	for _, rule := range rules.Rules {
		if r.Iface == nil || rule.Iface == nil || *r.Iface != *rule.Iface {
			continue
		}
		if (r.SourceIP != nil && rule.SourceIP == nil) || (r.SourceIP == nil && rule.SourceIP != nil) {
			continue
		}
		if (r.DestinationIP != nil && rule.DestinationIP == nil) || (r.DestinationIP == nil && rule.DestinationIP != nil) {
			continue
		}
		if (r.SourcePort != nil && rule.SourcePort == nil) || (r.SourcePort == nil && rule.SourcePort != nil) {
			continue
		}
		if (r.DestinationPort != nil && rule.DestinationPort == nil) || (r.DestinationPort == nil && rule.DestinationPort != nil) {
			continue
		}
		if r.SourceIP != nil && *r.SourceIP != *rule.SourceIP {
			continue
		}
		if r.DestinationIP != nil && *r.DestinationIP != *rule.DestinationIP {
			continue
		}
		if r.SourcePort != nil && *r.SourcePort != *rule.SourcePort {
			continue
		}
		if r.DestinationPort != nil && *r.DestinationPort != *rule.DestinationPort {
			continue
		}
		// we are here, the rule had been found, delete it
		out, err := exec.Command("tc", "filter", "del", "dev", *rule.Iface, "protocol", "ip", "parent", "1:0", "prio", "3", "handle", *rule.FilterHandle, "u32").CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %s", err, string(out))
		}
		break
	}

	return CleanupUnusedQdisc()
}
