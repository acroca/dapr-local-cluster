# Load Helm extension
load('ext://helm_resource', 'helm_resource', 'helm_repo')

helm_repo('bitnami', 'https://charts.bitnami.com/bitnami')
helm_resource('redis', 'bitnami/redis',
             namespace='default',
             flags=[
                '--set', 'architecture=standalone',
                '--set', 'auth.enabled=false',
                '--set', 'master.resources.requests.memory=512Mi',
                '--set', 'master.resources.requests.cpu=200m',
                '--set', 'master.resources.limits.memory=1024Mi',
                '--set', 'master.resources.limits.cpu=200m'
              ],
             resource_deps=['bitnami'],
             labels=['core'])

helm_repo('openzipkin', 'https://zipkin.io/zipkin-helm')
helm_resource('zipkin', 'openzipkin/zipkin',
             namespace='default',
             flags=['--set', 'zipkin.storage.type=mem'],
             resource_deps=['openzipkin'],
             labels=['core'],
             port_forwards=['9411:9411'])

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
  runtime_version = "latest"
  # runtime_version = "1.13.6"
  # runtime_version = "1.14.4"
  local_resource('dapr',
                cmd='''
                  mise exec dapr@%s -- dapr uninstall -k -n default && \
                  mise exec dapr@%s -- dapr init -k -n default --runtime-version %s --wait
                ''' % (dapr_version, dapr_version, runtime_version),
                labels=['core'])

k8s_kind('Configuration')
k8s_kind('Component')
k8s_yaml("manifests/config.yaml")
k8s_resource(workload='daprconfig', resource_deps=['dapr'], labels=['core'], pod_readiness="ignore")
k8s_yaml("manifests/component_pubsub.yaml")
k8s_resource(workload='pubsub', resource_deps=['dapr', 'redis'], labels=['core'], pod_readiness="ignore")
k8s_yaml("manifests/component_state.yaml")
k8s_resource(workload='statestore', resource_deps=['dapr', 'redis'], labels=['core'], pod_readiness="ignore")

load_dynamic('apps/pub/Tiltfile')
load_dynamic('apps/sub/Tiltfile')
load_dynamic('apps/workflows-py/Tiltfile')
load_dynamic('apps/workflows-go/Tiltfile')
# load_dynamic('apps/workflows-stress/Tiltfile')
