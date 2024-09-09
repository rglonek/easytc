package tc

import (
	"fmt"
	"os/exec"

	"github.com/bestmethod/inslice"
)

// remove all rules from a given interface; if interface is not given, removes all rules from all interfaces
func Reset(iface *string, verbose bool) error {
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
		comm := []string{"tc", "qdisc", "del", "dev", i, "root"}
		logf(verbose, "(Reset) Running %v", comm)
		exec.Command(comm[0], comm[1:]...).CombinedOutput()
	}
	return nil
}

func CleanupUnusedQdisc(verbose bool) error {
	rules, err := ListRules(verbose)
	if err != nil {
		return err
	}
	for _, rule := range rules.Rules {
		if rule.FilterHandle == nil && rule.Iface != nil && rule.QdiscHandle != nil && rule.FlowID != nil {
			comm := []string{"tc", "qdisc", "del", "dev", *rule.Iface, "parent", *rule.FlowID, "handle", *rule.QdiscHandle}
			logf(verbose, "(CleanupUnusedQdisc) Running %v", comm)
			out, err := exec.Command(comm[0], comm[1:]...).CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s:%s", err, string(out))
			}
		}
	}
	return nil
}

func Delete(r *Rule, verbose bool) error {
	// list qdisc
	rules, err := ListRules(verbose)
	if err != nil {
		return err
	}

	// list ifaces
	var ifaces []string
	if r.Iface == nil {
		ifaces = rules.Interfaces
	} else {
		ifaces = append(ifaces, *r.Iface)
		if !inslice.HasString(ifaces, *r.Iface) {
			return fmt.Errorf("interface %s does not exist", *r.Iface)
		}
	}

	for _, iface := range ifaces {
		for _, rule := range rules.Rules {
			if rule.Iface == nil || iface != *rule.Iface {
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
			comm := []string{"tc", "filter", "del", "dev", *rule.Iface, "protocol", "ip", "parent", "1:0", "prio", "3", "handle", *rule.FilterHandle, "u32"}
			logf(verbose, "(Delete) Running %v", comm)
			out, err := exec.Command(comm[0], comm[1:]...).CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s: %s", err, string(out))
			}
			break
		}
	}

	return CleanupUnusedQdisc(verbose)
}
