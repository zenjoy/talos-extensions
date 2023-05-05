# Add IP Rules extension

This is a small extension service to allow configuration of IP rules on Talos linux.

## Usage

Enable the extension in the machine configuration before installing Talos:

```yaml
machine:
  install:
    extensions:
      - image: ghcr.io/siderolabs/add-ip-rules:<VERSION>
```

And configure the extension via .machine.files. The file must be placed at
`/var/etc/add-ip-rules/config.yaml`:

```yaml
machine:
  files:
    - path: /var/etc/add-ip-rules/config.yaml
      permissions: 0o600
      op: create
      content: |-
        network:
          ethernets:
            ens3:
              routes:
                - to: 0.0.0.0/0
                  via: 192.168.3.1
                  metric: 100
                  table: 101
                - to: 1.2.3.4/28
                  via: 5.6.7.8
                  metric: 100
                  table: 102
              routing-policy:
                - from: 192.168.3.0/24
                  table: 101
                - from: 1.2.3.4/28
                  table: 102
```
