# add a user

1. user calls `just key-create`
2. admin adds user and public key to the `.sopy.yaml`

# edit a secret

Either manually

    export SOPS_AGE_KEY=$(security find-generic-password -a "$USER" -s "age-edkimo" -w)
    sops edit secrets/live/dotenv.env

or via just

    just sops edit secrets/test/dotenv.env

# check for differences

    just secrets diff

# apply secrets

    just secrets upload # dry run
    just secrets -doit upload

or

    just secrets manifest | kubectl apply --server-side -f -
