load('ext://helm_resource', 'helm_resource', 'helm_repo')

k8s_kind('Service', pod_readiness="ignore")
k8s_kind('Job', pod_readiness="ignore")

helm_repo('openzipkin', 'https://zipkin.io/zipkin-helm')
helm_repo('dapr-helm-repo', 'https://dapr.github.io/helm-charts')

run_e2e = False # False/True

pubsub_backend = 'kafka' # redis/kafka
state_backend = 'redis' # redis/postgres
use_oidc = True # True/False (only for kafka)

if run_e2e:
  load_dynamic('e2e/Tiltfile')
else:
  helm_resource('zipkin', 'openzipkin/zipkin',
              flags=['--set', 'zipkin.storage.type=mem'],
              resource_deps=['openzipkin'],
              labels=['core'],
              port_forwards=['9411:9411'])

  if state_backend == 'redis' or pubsub_backend == 'redis':
    k8s_yaml("manifests/redis.yaml")
    k8s_resource(workload='redis', labels=['core'])
    k8s_resource(workload='redis-master', labels=['core'])

    k8s_yaml("manifests/redis_insight.yaml")
    k8s_resource(workload='redisinsight', resource_deps=['redis'], labels=['core'], port_forwards=['5540:5540'], links="http://localhost:5540/0/browser")

  if state_backend == 'postgres':
    k8s_yaml("manifests/postgres.yaml")
    k8s_resource(workload='postgres', objects=['postgres-config:ConfigMap:default'], labels=['core'])

  if pubsub_backend == 'kafka':
    if use_oidc:
      # Deploy Strimzi Kafka Operator (for OAuth/OIDC support) and Keycloak IdP
      helm_repo('strimzi', 'https://strimzi.io/charts/')
      helm_resource('strimzi-kafka-operator', 'strimzi/strimzi-kafka-operator',
                    flags=['--set', 'watchAnyNamespace=true'],
                    resource_deps=['strimzi'],
                    labels=['core'])

      k8s_yaml("manifests/kafka-oidc.yaml")
      k8s_resource(objects=['keycloak-realm-local:ConfigMap:default'], new_name='keycloak-cm', labels=['core'])
      k8s_resource(workload='keycloak:deployment', resource_deps=['keycloak-cm'], labels=['core'])
      k8s_resource(workload='keycloak-bootstrap', resource_deps=['keycloak:deployment'], labels=['core'])
      k8s_resource(workload='keycloak:service', resource_deps=['keycloak:deployment'], labels=['core'])
      k8s_resource(objects=['kafka:kafka', 'kafka-pool:KafkaNodePool:default'], new_name='kafka', resource_deps=['strimzi-kafka-operator', 'keycloak:deployment'], labels=['core'])
    else:
      k8s_yaml("manifests/kafka.yaml")
      k8s_resource(workload='kafka', labels=['core'])

  dapr_cli_version = "1.15"
  # dapr_cli_version = "dev" # use ../dapr instead of a release

  if dapr_cli_version == "dev":
    local_resource('dapr-images',
                  dir='../dapr',
                  cmd='mise exec dapr@1.15 -- dapr uninstall -k && make build docker-push',
                  env={
                    'HA_MODE': 'true',
                    'DAPR_REGISTRY': 'localhost:5001/dapr',
                    'DAPR_TAG': 'dev',
                    'DAPR_TEST_NAMESPACE': 'default',
                    'DAPR_NAMESPACE': 'default',
                    'TARGET_OS': 'linux',
                    'TARGET_ARCH': 'arm64',
                    'GOOS': 'linux',
                    'GOARCH': 'arm64',
                    'LOG_LEVEL': 'debug'
                  },
                  labels=['core'])

    helm_resource('dapr', '../dapr/charts/dapr',
                release_name='dapr',
                flags=[
                    '--set', 'global.ha.enabled=true',
                    '--set', 'global.tag=dev-linux-arm64',
                    '--set', 'global.registry=localhost:5001/dapr',
                    '--set', 'global.logAsJson=true',
                    '--set', 'global.daprControlPlaneOs=linux',
                    '--set', 'global.daprControlPlaneArch=arm64',
                    '--set', 'dapr_placement.logLevel=debug',
                    '--set', 'dapr_sentry.logLevel=debug',
                    '--set', 'dapr_sidecar_injector.sidecarImagePullPolicy=Always',
                    '--set', 'global.imagePullPolicy=Always',
                    '--set', 'global.imagePullSecrets=',
                    '--set', 'global.mtls.enabled=true',
                    '--set', 'dapr_placement.cluster.forceInMemoryLog=true',
                    '--set', 'dapr_scheduler.replicaCount=1',
                    '--set', 'dapr_scheduler.cluster.storageSize=100Mi',
                    '--set', 'dapr_scheduler.cluster.inMemoryStorage=false',
                    '--set', 'dapr_scheduler.logLevel=debug',
                  ],
                resource_deps=['dapr-images'],
                labels=['core'])
  else:
    helm_resource('dapr', 'dapr/dapr',
      release_name='dapr',
      flags=[
        '--set', 'global.ha.enabled=true',
        '--set', 'global.mtls.enabled=true',
        '--version', "1.16.0",
      ],
      resource_deps=['dapr-helm-repo'],
      labels=['core'])

  pubsub_component = pubsub_backend
  if use_oidc:
    pubsub_component = "kafka-oidc"
  k8s_yaml("manifests/component_config.yaml")
  k8s_yaml("manifests/component_pubsub.%s.yaml" % pubsub_component)
  k8s_yaml("manifests/component_state.%s.yaml" % state_backend)
  k8s_yaml("manifests/component_workflowstate.%s.yaml" % state_backend)
  k8s_resource(
    objects=['daprconfig', 'pubsub', 'statestore', 'workflowstatestore'],
    new_name='components',
    resource_deps=['dapr', state_backend, pubsub_backend],
    labels=['core'],
    pod_readiness="ignore"
  )


  # load_dynamic('apps/actors-go/Tiltfile')
  load_dynamic('apps/pub/Tiltfile')
  load_dynamic('apps/sub/Tiltfile')
  # load_dynamic('apps/workflows-py/Tiltfile')
  # load_dynamic('apps/workflows-crossapp/Tiltfile')
  # load_dynamic('apps/workflows-go/Tiltfile')
  # load_dynamic('apps/workflows-stress/Tiltfile')
  # load_dynamic('apps/dapr-agents/Tiltfile')
  # load_dynamic('apps/tracing-dotnet/Tiltfile')



