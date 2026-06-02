# Local Docker Test Services

These commands are for manual functional testing after node TCP access paths are implemented. Test resources use the `oneproxy-test-` prefix and can be removed without affecting production services.

```bash
docker network create oneproxy-test-net

docker run -d --name oneproxy-test-ssh --network oneproxy-test-net \
  -e PASSWORD_ACCESS=true \
  -e USER_NAME=ubuntu \
  -e USER_PASSWORD=ubuntu \
  linuxserver/openssh-server:latest

docker run -d --name oneproxy-test-tls --network oneproxy-test-net nginx:alpine
```

Smoke checks:

```bash
docker exec oneproxy-test-ssh nc -z 127.0.0.1 2222
docker exec oneproxy-test-tls nginx -t
```

Cleanup:

```bash
docker rm -f oneproxy-test-ssh oneproxy-test-tls
docker network rm oneproxy-test-net
```
