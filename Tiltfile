# Load Helm extension
load('ext://helm_resource', 'helm_resource', 'helm_repo')

# Redis deployment using Helm
helm_repo('bitnami', 'https://charts.bitnami.com/bitnami')
helm_resource('redis', 'bitnami/redis',
             namespace='default',
             flags=['--set', 'auth.enabled=false'],
             labels=['core'])


dapr_version = "dev"

if dapr_version == "dev":
  local_resource('dapr',
                dir='../dapr',
                cmd='mise exec dapr@1.15 -- dapr uninstall -k -n default && make build docker-push docker-deploy-k8s',
                env={
                  'HA_MODE': 'true',
                  'DAPR_REGISTRY': 'localhost:5001/dapr',
                  'DAPR_TAG': 'dev',
                  'DAPR_TEST_NAMESPACE': 'dapr-tests',
                  'DAPR_NAMESPACE': 'default',
                  'TARGET_OS': 'linux',
                  'TARGET_ARCH': 'arm64',
                  'GOOS': 'linux',
                  'GOARCH': 'arm64',
                  'LOG_LEVEL': 'debug'
                },
                labels=['core'])
else:
  local_resource('dapr',
                cmd='''
                  mise exec dapr@%s -- dapr uninstall -k -n default && \
                  mise exec dapr@%s -- dapr init -k -n default
                ''' % (dapr_version, dapr_version),
                labels=['core'])

k8s_kind('Component')
k8s_yaml("manifests/component_pubsub.yaml")
k8s_resource(workload='pubsub', resource_deps=['dapr', 'redis'], labels=['core'], pod_readiness="ignore")
k8s_yaml("manifests/component_state.yaml")
k8s_resource(workload='statestore', resource_deps=['dapr', 'redis'], labels=['core'], pod_readiness="ignore")

load_dynamic('apps/pub/Tiltfile')
load_dynamic('apps/sub/Tiltfile')
load_dynamic('apps/workflows-py/Tiltfile')
load_dynamic('apps/workflows-go/Tiltfile')
