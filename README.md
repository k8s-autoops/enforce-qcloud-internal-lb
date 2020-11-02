# enforce-qcloud-internal-lb

自动强制为 Loadbalancer 类型的 Service 切换为内网类型的负载均衡

## 使用方式

本组件使用 `admission-bootstrapper` 安装，首先参照此文档 https://github.com/k8s-autoops/admission-bootstrapper ，完成 `admission-bootstrapper` 的初始化步骤。

然后，部署以下 YAML 即可

```yaml
# create serviceaccount
apiVersion: v1
kind: ServiceAccount
metadata:
  name: enforce-qcloud-internal-lb
  namespace: autoops
---
# create clusterrole
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: enforce-qcloud-internal-lb
rules:
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["get"]
---
# create clusterrolebinding
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: enforce-qcloud-internal-lb
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: enforce-qcloud-internal-lb
subjects:
  - kind: ServiceAccount
    name: enforce-qcloud-internal-lb
    namespace: autoops
---
# create job
apiVersion: batch/v1
kind: Job
metadata:
  name: install-enforce-qcloud-internal-lb
  namespace: autoops
spec:
  template:
    spec:
      serviceAccount: admission-bootstrapper
      containers:
        - name: admission-bootstrapper
          image: autoops/admission-bootstrapper
          env:
            - name: ADMISSION_NAME
              value: enforce-qcloud-internal-lb
            - name: ADMISSION_IMAGE
              value: autoops/enforce-qcloud-internal-lb
            - name: ADMISSION_ENVS
              value: ""
            - name: ADMISSION_SERVICE_ACCOUNT
              value: "enforce-qcloud-internal-lb"
            - name: ADMISSION_MUTATING
              value: "true"
            - name: ADMISSION_IGNORE_FAILURE
              value: "false"
            - name: ADMISSION_SIDE_EFFECT
              value: "None"
            - name: ADMISSION_RULES
              value: '[{"operations":["CREATE"],"apiGroups":[""], "apiVersions":["*"], "resources":["services"]}]'
      restartPolicy: OnFailure
```

最后为需要启用的命名空间，添加注解 `autoops.enforce-qcloud-internal-lb/subnet=subnet-xxxxxx`

## Credits

Guo Y.K., MIT License
