[![CircleCI](https://circleci.com/gh/giantswarm/template.svg?style=shield)](https://circleci.com/gh/giantswarm/template)

# REPOSITORY_NAME

This is a template repository containing some basic files every repository
needs.

To use it just hit `Use this template` button or [this link][generate].

Things to do with your newly created repo:

1. Run`devctl replace -i "REPOSITORY_NAME" "$(basename $(git rev-parse
   --show-toplevel))" --ignore '.git/**' '**'`.
2. Run `devctl replace -i "template" "$(basename $(git rev-parse
   --show-toplevel))" --ignore '.git/**' '**'`.
3. Go to https://github.com/giantswarm/REPOSITORY_NAME/settings and make sure `Allow
   merge commits` box is unchecked and `Automatically delete head branches` box
   is checked.
4. Go to https://github.com/giantswarm/REPOSITORY_NAME/settings/access and add
   `giantswarm/bots` with `Write` access and `giantswarm/employees` with
   `Admin` access.
5. Add this repository to https://github.com/giantswarm/github.
6. Create quay.io docker repository if needed.
7. Add the project to the CircleCI:
   https://circleci.com/setup-project/gh/giantswarm/REPOSITORY_NAME
8. Change the badge (with style=shield):
   https://circleci.com/gh/giantswarm/REPOSITORY_NAME.svg?style=shield&circle-token=TOKEN_FOR_PRIVATE_REPO
   If this is a private repository token with scope `status` will be needed.

[generate]: https://github.com/giantswarm/template/generate
