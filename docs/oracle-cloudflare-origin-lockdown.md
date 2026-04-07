# Oracle Origin Lockdown With Cloudflare

This runbook locks the Oracle host so public web traffic can reach the
application only through Cloudflare. Direct public access to the Oracle origin
IP must fail. SSH remains available only from the administrator network, and app
containers remain reachable from the host through localhost or Docker's private
network.

## Target Policy

- Public users reach `https://aitravel.dpdns.org` through Cloudflare only.
- The Oracle public IP does not accept direct public TCP `80` or `443` traffic
  from arbitrary source IPs.
- SSH TCP `22` is allowed only from the administrator CIDR, for example
  `<your-admin-ip>/32`.
- Local host checks may use `127.0.0.1`.
- Docker-internal app traffic continues on the Compose network, for example
  `web -> api:8080`.

## Recommended Option: Cloudflare Tunnel

Use Cloudflare Tunnel when possible. It avoids exposing Oracle TCP `80` and
`443` at all, because `cloudflared` opens an outbound tunnel to Cloudflare and
Cloudflare routes the public hostname through that tunnel.

### 1. Cloudflare DNS

1. In Cloudflare, create a tunnel for this application.
2. Route `aitravel.dpdns.org` to the tunnel.
3. Remove any DNS-only `A`, `AAAA`, or `CNAME` records that reveal the Oracle
   origin IP.

### 2. Oracle Cloud ingress

In the OCI Console:

1. Go to `Networking` -> `Virtual cloud networks`.
2. Open the VCN used by the Oracle compute instance.
3. Prefer `Network Security Groups`; if the instance is not in an NSG, update
   the subnet `Security Lists` instead.
4. Remove ingress rules for:
   - source `0.0.0.0/0`, protocol TCP, port `80`
   - source `0.0.0.0/0`, protocol TCP, port `443`
   - source `::/0`, protocol TCP, port `80`
   - source `::/0`, protocol TCP, port `443`
5. Keep or add only:
   - source `<your-admin-ip>/32`, protocol TCP, port `22`
   - IPv6 equivalent only if the admin network really uses IPv6

Do not add Cloudflare IP ranges for `80` or `443` in the Tunnel model.

### 3. Bind the web container to localhost

When a local host process such as `cloudflared` connects to the Compose web
container, publish the container on loopback only:

```yaml
web:
  ports:
    - "127.0.0.1:${WEB_PORT:-8080}:80"
```

Then point the tunnel to the local published port:

```yaml
ingress:
  - hostname: aitravel.dpdns.org
    service: http://127.0.0.1:8080
  - service: http_status:404
```

### 4. Host firewall

Close public web ports on the Oracle host firewall:

```bash
sudo firewall-cmd --permanent --zone=public --remove-service=http || true
sudo firewall-cmd --permanent --zone=public --remove-service=https || true
sudo firewall-cmd --permanent --zone=public --remove-port=80/tcp || true
sudo firewall-cmd --permanent --zone=public --remove-port=443/tcp || true
sudo firewall-cmd --reload
```

Allow SSH only from the administrator CIDR:

```bash
ADMIN_CIDR="<your-admin-ip>/32"

sudo firewall-cmd --permanent --zone=public --remove-service=ssh || true
sudo firewall-cmd --permanent --zone=public \
  --add-rich-rule="rule family=ipv4 source address=\"${ADMIN_CIDR}\" service name=\"ssh\" accept"
sudo firewall-cmd --reload
```

If the host is not using `firewalld`, apply the same policy in the active
firewall manager before closing the OCI ingress rules.

## Alternative Option: Cloudflare Proxied DNS To Origin

Use this only if Cloudflare must connect directly to the Oracle public IP. In
this model, the Oracle host accepts TCP `443` only from Cloudflare's published
IP ranges.

### 1. Cloudflare DNS

1. In Cloudflare DNS, confirm `aitravel.dpdns.org` is `Proxied`.
2. Remove DNS-only records that reveal or point to the same origin.
3. Do not publish alternate hostnames that bypass Cloudflare.

### 2. Download the current Cloudflare IP ranges

Cloudflare IP ranges change over time. Pull them at the time of the firewall
change instead of copying a stale list into this repository.

```bash
curl -fsS https://www.cloudflare.com/ips-v4 -o /tmp/cloudflare-ips-v4.txt
curl -fsS https://www.cloudflare.com/ips-v6 -o /tmp/cloudflare-ips-v6.txt
```

### 3. Oracle Cloud ingress

In the OCI Console:

1. Open the instance's NSG, or the subnet Security List if NSGs are not used.
2. Remove broad web ingress:
   - source `0.0.0.0/0`, TCP `80`
   - source `0.0.0.0/0`, TCP `443`
   - source `::/0`, TCP `80`
   - source `::/0`, TCP `443`
3. Add ingress rules for every CIDR in `/tmp/cloudflare-ips-v4.txt`:
   - source `<cloudflare-ipv4-cidr>`, TCP `443`
4. Add ingress rules for every CIDR in `/tmp/cloudflare-ips-v6.txt` only if the
   instance has IPv6 enabled:
   - source `<cloudflare-ipv6-cidr>`, TCP `443`
5. Add TCP `80` from Cloudflare IPs only if the origin still needs HTTP
   redirect traffic. Prefer Cloudflare "Always Use HTTPS" and close origin
   TCP `80`.
6. Keep SSH TCP `22` limited to `<your-admin-ip>/32`.

OCI evaluates NSG and Security List rules together. If both are attached, make
sure neither layer still allows broad public `80` or `443` ingress.

### 4. Host firewall Cloudflare allowlist

Mirror the OCI policy on the host. The exact firewall tool can vary by image;
these commands assume `firewalld`.

```bash
sudo firewall-cmd --permanent --zone=public --remove-service=http || true
sudo firewall-cmd --permanent --zone=public --remove-service=https || true
sudo firewall-cmd --permanent --zone=public --remove-port=80/tcp || true
sudo firewall-cmd --permanent --zone=public --remove-port=443/tcp || true

while read -r cidr; do
  [ -n "$cidr" ] || continue
  sudo firewall-cmd --permanent --zone=public \
    --add-rich-rule="rule family=ipv4 source address=\"${cidr}\" port protocol=\"tcp\" port=\"443\" accept"
done < /tmp/cloudflare-ips-v4.txt

while read -r cidr; do
  [ -n "$cidr" ] || continue
  sudo firewall-cmd --permanent --zone=public \
    --add-rich-rule="rule family=ipv6 source address=\"${cidr}\" port protocol=\"tcp\" port=\"443\" accept"
done < /tmp/cloudflare-ips-v6.txt

ADMIN_CIDR="<your-admin-ip>/32"
sudo firewall-cmd --permanent --zone=public --remove-service=ssh || true
sudo firewall-cmd --permanent --zone=public \
  --add-rich-rule="rule family=ipv4 source address=\"${ADMIN_CIDR}\" service name=\"ssh\" accept"

sudo firewall-cmd --reload
```

### 5. Keep containers local behind host nginx

If host nginx terminates TLS on TCP `443`, the Docker web container should not
publish TCP `80` to every interface. Bind it to loopback:

```yaml
web:
  ports:
    - "127.0.0.1:${WEB_PORT:-8080}:80"
```

The host nginx upstream should then point to localhost:

```nginx
location / {
  proxy_pass http://127.0.0.1:8080;
  proxy_set_header Host $host;
  proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
  proxy_set_header X-Forwarded-Proto https;
}
```

The app's internal nginx can continue proxying `/api/` to `http://api:8080/api/`
on the Docker Compose network.

## Validation

From the Oracle host:

```bash
curl -fsS http://127.0.0.1:8080/healthz
docker compose --project-directory /home/opc/apps/work-tree-ex ps
docker compose --project-directory /home/opc/apps/work-tree-ex logs --tail=100 web
docker compose --project-directory /home/opc/apps/work-tree-ex logs --tail=100 api
```

From a normal external network:

```bash
curl -I https://aitravel.dpdns.org/
curl -I https://aitravel.dpdns.org/healthz
```

The hostname should work through Cloudflare. Direct origin IP checks for TCP
`80` or `443` should fail after the firewall changes. Do not rely on browser
tests alone; confirm with OCI rules and the host firewall state.

## Maintenance

- Re-sync Cloudflare IP ranges on a schedule if using the proxied DNS model.
- Prefer Cloudflare Tunnel to avoid maintaining IP allowlists.
- Keep SSH CIDRs narrow and rotate deploy keys separately from personal SSH
  keys.
- After any deployment change, verify `/healthz` through the Cloudflare
  hostname and from localhost on the Oracle host.

## References

- Cloudflare IP ranges: https://www.cloudflare.com/ips/
- Cloudflare origin protection: https://developers.cloudflare.com/fundamentals/security/protect-your-origin-server/
- Cloudflare Tunnel ingress: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/configure-tunnels/local-management/configuration-file/
- OCI security rules: https://docs.oracle.com/iaas/Content/Network/Concepts/securityrules.htm
