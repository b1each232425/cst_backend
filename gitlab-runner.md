```bash
runner-shell register \
 --url https://git.w2w.me:6443 \
 --registration-token glrt-TAy1yeoYyqLZxfoJk9BQ \
 --docker-privileged \
 --executor docker \
 --description "alpine-ws" \
 --docker-image "alpine:ws" \
 --docker-pull-policy if-not-present \
 --docker-volumes /var/run/docker.sock:/var/run/docker.sock \
 --docker-volumes data:/var/data \
 --docker-volumes deploy:/var/deploy

pnpm config set store-dir /path/to/.pnpm-store

```