# enforce-qcloud-internal-lb

自动强制为 腾讯云 TKE 集群 Loadbalancer 类型的 Service 切换为内网类型的负载均衡

## 使用方式

* 初始化 `admission-bootstrapper` 
  参照此文档 https://github.com/k8s-autoops/admission-bootstrapper ，完成 `admission-bootstrapper` 的初始化步骤
* 部署以下 YAML

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

* 为需要启用的命名空间，添加注解，指明要使用的内网

  `autoops.enforce-qcloud-internal-lb/subnet=subnet-xxxxxx`
  
  **可以配合 `enforce-ns-annotations` 自动为新命名空间启用此注解**

## Credits

Guo Y.K., MIT License
