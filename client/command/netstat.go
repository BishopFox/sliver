package command

// func netstat(ctx *grumble.Context, rpc RPCServer) {
// 	if ActiveSliver.Sliver == nil {
// 		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
// 		return
// 	}

// 	listening := ctx.Flags.Bool("listen")
// 	ip4 := ctx.Flags.Bool("ip4")
// 	ip6 := ctx.Flags.Bool("ip6")
// 	tcp := ctx.Flags.Bool("tcp")
// 	udp := ctx.Flags.Bool("udp")

// 	data, _ := proto.Marshal(&sliverpb.NetstatRequest{
// 		SliverID:  ActiveSliver.Sliver.ID,
// 		TCP:       tcp,
// 		UDP:       udp,
// 		Listening: listening,
// 		IP4:       ip4,
// 		IP6:       ip6,
// 	})
// 	resp := <-rpc(&sliverpb.Envelope{
// 		Type: sliverpb.MsgNetstatReq,
// 		Data: data,
// 	}, defaultTimeout)
// 	if resp.Err != "" {
// 		fmt.Printf(Warn+"Error: %s", resp.Err)
// 		return
// 	}
// 	netstatResp := &sliverpb.NetstatResponse{}
// 	err := proto.Unmarshal(resp.Data, netstatResp)
// 	if err != nil {
// 		fmt.Printf(Warn + "Failed to decode response\n")
// 		return
// 	}
// 	displayEntries(netstatResp.Entries)
// }

// func displayEntries(entries []*sliverpb.SockTabEntry) {
// 	lookup := func(skaddr *sliverpb.SockTabEntry_SockAddr) string {
// 		const IPv4Strlen = 17
// 		addr := skaddr.Ip
// 		names, err := net.LookupAddr(addr)
// 		if err == nil && len(names) > 0 {
// 			addr = names[0]
// 		}
// 		if len(addr) > IPv4Strlen {
// 			addr = addr[:IPv4Strlen]
// 		}
// 		return fmt.Sprintf("%s:%d", addr, skaddr.Port)
// 	}

// 	fmt.Printf("Proto %-23s %-23s %-12s %-16s\n", "Local Addr", "Foreign Addr", "State", "PID/Program name")
// 	for _, e := range entries {
// 		p := ""
// 		if e.Proc != nil {
// 			p = fmt.Sprintf("%d/%s", e.Proc.Pid, e.Proc.Executable)
// 		}
// 		saddr := lookup(e.LocalAddr)
// 		daddr := lookup(e.RemoteAddr)
// 		if e.Proc != nil && e.Proc.Pid == ActiveSliver.Sliver.PID && isSliverAddr(daddr) {
// 			fmt.Printf("%s%-5s %-23.23s %-23.23s %-12s %-16s%s\n", green, e.Proto, saddr, daddr, e.SkState, p, normal)
// 		} else {

// 			fmt.Printf("%-5s %-23.23s %-23.23s %-12s %-16s\n", e.Proto, saddr, daddr, e.SkState, p)
// 		}
// 	}
// }

// func isSliverAddr(daddr string) bool {
// 	parts := strings.Split(daddr, ":")
// 	if len(parts) != 3 {
// 		return false
// 	}
// 	c2Addr := strings.Split(ActiveSliver.Sliver.ActiveC2, "://")[1]
// 	return strings.Join(parts[:2], ":") == c2Addr
// }
