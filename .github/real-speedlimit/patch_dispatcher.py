#!/usr/bin/env python3
from pathlib import Path

p = Path('app/dispatcher/default.go')
s = p.read_text()

def rep(old, new):
    global s
    if old not in s:
        raise SystemExit('anchor not found')
    s = s.replace(old, new, 1)

rep('\t"github.com/xtls/xray-core/common"\n', '\t"github.com/xtls/xray-core/app/limiter"\n\t"github.com/xtls/xray-core/common"\n')
rep('var errSniffingTimeout = errors.New("timeout on sniffing")\n', 'var errSniffingTimeout = errors.New("timeout on sniffing")\n\nvar clientRateLimiters = limiter.NewManager()\n')
rep('\tif user != nil && len(user.Email) > 0 {\n\t\tp := d.policy.ForLevel(user.Level)\n', '\tif user != nil && len(user.Email) > 0 {\n\t\trates := clientRateLimiters.ForUser(user.Email, user.SpeedLimitUpMbps, user.SpeedLimitDownMbps)\n\t\tinboundLink.Writer = limiter.WrapWriter(ctx, inboundLink.Writer, rates.Up)\n\t\toutboundLink.Writer = limiter.WrapWriter(ctx, outboundLink.Writer, rates.Down)\n\n\t\tp := d.policy.ForLevel(user.Level)\n')
rep('\tlink.Reader = &buf.TimeoutWrapperReader{Reader: link.Reader}\n\n\tif user != nil && len(user.Email) > 0 {\n\t\tp := policyManager.ForLevel(user.Level)\n', '\tif user != nil && len(user.Email) > 0 {\n\t\trates := clientRateLimiters.ForUser(user.Email, user.SpeedLimitUpMbps, user.SpeedLimitDownMbps)\n\t\tlink.Reader = limiter.WrapReader(ctx, link.Reader, rates.Up)\n\t\tlink.Writer = limiter.WrapWriter(ctx, link.Writer, rates.Down)\n\t}\n\n\tlink.Reader = &buf.TimeoutWrapperReader{Reader: link.Reader}\n\n\tif user != nil && len(user.Email) > 0 {\n\t\tp := policyManager.ForLevel(user.Level)\n')
p.write_text(s)
