# =============================================================================
# replit.nix — additional system packages for the AVEX platform
# =============================================================================
# These are installed via Nix on top of the modules declared in .replit
# (nodejs-20, postgresql-16, go-1.25, bash, web).
#
# We need:
#   - redis: for the realtime + outbox modules (caching, pub/sub, jobs queue)
#   - curl: for health checks in run.sh
#   - jq: for parsing JSON responses in seed-driver.sh
#   - gettext: for envsubst (used by start scripts if needed)
# =============================================================================
{pkgs}: {
  deps = [
    pkgs.redis
    pkgs.curl
    pkgs.jq
    pkgs.gettext
  ];
}
