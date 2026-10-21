### Release Process

Releases are created from a candidate branch.  Create a candidate branch by running:

```bash
$ make init_release
```

Follow the prompts.

Once the release is ready to be tagged execute the publish flow:

```bash
$ make publish_release
```

Follow the prompts to tag either a `beta` pre-release set of tags or a `stable` tag without a pre-release suffix.

The following packages will be tagged:
- generator/protoc-gen-cre
- capabilities/scheduler/cron
- capabilities/networking/http
- capabilities/blockchain/evm

Merge and delete the release branch once tags are accepted and verified.