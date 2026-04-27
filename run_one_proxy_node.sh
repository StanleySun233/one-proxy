cd ./apps/one-proxy-node

if [ -z "$NODE_JOIN_PASSWORD" ]; then
  NODE_JOIN_PASSWORD='password'
fi

PATH="$HOME/.local/go/bin:$HOME/go/bin:$PATH" \
  NODE_JOIN_PASSWORD="$NODE_JOIN_PASSWORD" \
  air
