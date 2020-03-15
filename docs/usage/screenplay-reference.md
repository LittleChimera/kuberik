# Screenplay reference

## Screenplay

### Variables

To define variables shared by all jobs, define the variables field on the screenplay. You can see a detailed definition of each variable in [Vars section of API reference](./screenplay-fields.md#variable). These variables are available in all jobs as environment variables and are also mounted as file under `/kuberik` in every container.

```yaml
screenplay:
  variables:
  - name: foo
    value: bar
```

### Provisioned volumes

In some cases, you'd want to use a temporary storage to share information between the jobs you're executing. To do so, you can define a volume claim template.

```yaml
screenplay:
  volumeClaimTemplates:
  - metadata:
      name: shared
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 1Gi
```

To use the provisioned volume in your Play, define a [volume mount][VolumeMount] in your containers.

```yaml
containers:
- volumeMounts:
  - name: shared
    mountPath: /shared
```

### Scenes

To define workloads which need to be executed one after another, add them to the array named `scenes`.

```yaml
screenplay:
  scenes:
    - name: scene_1
      ...
    - name: scene_2
      ...
```

## Scene

To define workloads which need to be executed in parallel, add them to the array named `frames` inside of a `scene` object.

```yaml
scenes:
  - name: scene_1
    frames:
      - name: frame_1
        ...
      - name: frame_2
        ...
```

## Frame

### Command and arguments
```yaml
vars:
- name: FOO
  value: bar
...
frames:
  - name: frame_1
    action:
      ...
      imagePullSecret: [{"name": "my-secret"}]
      .
```

### Image

### Image pull secrets

If you're running a container with an image from a private repository, you'll likely need to specify an image pull secret so that the Docker daemon can authenticate against your private repository and pull the requested image. Add `imagePullSecret` field in [PodSpec] to authenticate against a private registry when pulling the image.

```yaml
frames:
  - name: frame_1
    action:
      ...
      imagePullSecret: [{"name": "my-secret"}]
      ...
```

### Retrying

To improve the resiliency of your pipelines, it is recommended to add retries on every frame. Keep in mind though, that in that case, pipelines need to be [idempotent](https://en.wikipedia.org/wiki/Idempotence). Retrying a frame is achievable by using `backOffLimit` field from [JobSpec].

```yaml
frames:
  - name: retry-me
    action:
      backOffLimit: 5
    ...
```

### Copies

Copies enable you to spawn multiple instances of the same task so that the pipeline can allocate dynamic resources. To identify tasks, you can use the `FRAME_COPY_ID` environment variable. Every task in a loop will get an unique ordered index number.

```yaml
frames:
  - name: positions
    loop: 3
    action:
      ...
      command: ["echo", "I am on position number $(FRAME_COPY_ID)"]
    ...
```

### Skipping frames

Frames can be skipped using the `when` field.

### Ignoring errors

Errors can be ignored per frame or per scene. By default, errors are not ignored.

```yaml{3,6}
scenes:
  - name: scene_1
    ignoreErrors: true
    frames:
      - name: frame_1
        ignoreErrors: true
        ...
```

[JobSpec]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#jobspec-v1-batch
[PodSpec]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#podspec-v1-core
[VolumeMount]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#volumemount-v1-core
