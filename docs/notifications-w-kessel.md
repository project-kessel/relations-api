# Deploying Notifications with Relations API and Inventory API

## Bonfire

Notifications already lists Relations API and Inventory API as a dependency, so deploying all three services is made much easier with Bonfire

```shell
bonfire deploy notifications --source=appsre --ref-env insights-stage --timeout 600
```

## Testing Changes with Bonfire

The [Ephemeral Testing](./ephemeral-testing.md) doc already talks about how to test code changes in ephemeral using a local config. Testing changes with notifications can be done the same way, but its important to call out that the Notifications ClowdApp defines Relations API and Inventory API as a dependency. That means both services will be deployed along with Notifications. This is where bonfire can really shine!

If you need to test multiple changes at once (say you've made changes to Inventory API and Notifications), bonfire will take into consideration the settings in your file for any dependent apps. Meaning, since Notifications defines Inventory as a dependency, if you have local config settings for Inventory in your bonfire config, bonfire will deploy those specific settings including the image version.

```yaml
# Sample local config that defines notifications and inventory
apps:
- name: notifications
  components:
    - name: notifications-backend
      host: local
      repo: /path/to/notifications-backend-cloned-repo
      path: .rhcicd/clowdapp-backend.yaml
- name: kessel
  components:
    - name: kessel-inventory
      host: local
      repo: /path/to/inventory-api-cloned-repo
      path: deploy/kessel-inventory-ephem.yaml
      parameters:
        RELATIONS_IMAGE: quay.io/my-repo/inventory-api
        RELATIONS_IMAGE_TAG: a1B2c3
```

Using the above example, when deploying notifications using local config
1) bonfire will deploy the template using your specific local version and parameters
2) bonfire will detect the dependencies listed in the Clowdapp for notifications which includes inventory-api
3) when deploying inventory-api, the local config settings will be taken instead
