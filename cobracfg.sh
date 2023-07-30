go install github.com/spf13/cobra-cli@latest

cat > $HOME/.cobra.yaml << EOF
author: Nick Wright <nwright970@gmail.com>
license: MIT
useViper: true
EOF

cat $HOME/.cobra.yaml
