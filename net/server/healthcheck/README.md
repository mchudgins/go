# healthcheck

This is a modified version of github.com/heptiolabs/healthcheck.

It has been modified to support passing a context.Context to the liveness/readiness checks.
Passing the context will support tracing of a check as it passes thru the entire system.
