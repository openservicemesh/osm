# https://github.com/tilt-dev/tilt-extensions/tree/master/restart_process
load('ext://restart_process', 'docker_build_with_restart')

config.define_string_list("osm-values", args=True)
cfg = config.parse()

# This configures only the following resources. This way helm doesn't deploy
# cleanup and pre-install jobs
# https://docs.tilt.dev/tiltfile_config.html#run-a-defined-set-of-services
resources = ["osm-bootstrap",
    "osm-controller",
    "osm-injector",
    "controller-compile",
    "injector-compile",
    "bootstrap-compile",
    "wasm-compile",
    "demo"]
config.set_enabled_resources(resources)

# The default charts have a restricted security context
# we remove the security context from the charts to enable
# this is done here instead of the helm chart because its a developer tool not something
# we want to disable for general use
def remove_securityContext(yaml_stream):
    for item in yaml_stream:
        if item["kind"] == "Deployment":
            item['spec']['template']['spec']['securityContext'] = None

    return yaml_stream

def dockerfile_gen(binary_name):
    tilt_dockerfile_header = """
FROM gcr.io/distroless/base:debug as tilt
WORKDIR /
COPY ./{} /{}
""".format(binary_name, binary_name)

    return "\n".join([
        tilt_dockerfile_header,
    ])

local("kubectl create ns osm-system || true", quiet = True)

osm_values=[ 'image.tag=latest-main',
       'osm.controllerLogLevel=trace']
# load user settings
extra_values=cfg.get("osm-values", [])
osm_values.extend(extra_values)
yaml = helm(
  'charts/osm',
  name='osm-dev',
  namespace='osm-system',
  # Values to set from the command-line
  set=osm_values,
)

yaml_dict = remove_securityContext(decode_yaml_stream(yaml))
k8s_yaml((encode_yaml_stream(yaml_dict)))

# compile binaries
local_resource(
  'controller-compile',
  'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./build/osm-controller ./cmd/osm-controller',
  deps=['./cmd/osm-controller/osm-controller.go', 'pkg'],
  labels=['compile'],
)

local_resource(
  'injector-compile',
  'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./build/osm-injector  ./cmd/osm-injector',
  deps=['./cmd/osm-injector/osm-injector.go', 'pkg'],
  labels=['compile']
)

local_resource(
  'bootstrap-compile',
  'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./build/osm-bootstrap ./cmd/osm-bootstrap',
  deps=['./cmd/osm-bootstrap/osm-bootstrap.go', 'pkg'],
  labels=['compile']
)

wasm_compile = """
make buildx-context && \
docker buildx build \
    --builder osm --platform=linux/amd64 \
    -o type=docker --build-arg GO_BASE_IMAGE=notused \
    --build-arg FINAL_BASE_IMAGE=notused --target wasm \
    -t osm-wasm -f dockerfiles/Dockerfile.osm-controller . && \\
    docker run -v $PWD/pkg/envoy/lds:/updatefolder osm-wasm \
    cp /wasm/stats.wasm /updatefolder
"""
local_resource(
  'wasm-compile',
  wasm_compile,
  labels=['compile'],
  deps=['./wasm']
)

# wire up redeploys
# https://docs.tilt.dev/example_go.html#step-3-lets-live-update-it
docker_build_with_restart(
    'openservicemesh/osm-controller',
    context='./build/',
    target='tilt',
    dockerfile_contents = dockerfile_gen('osm-controller'),
    only=[
        'osm-controller'
    ],
    entrypoint=['/osm-controller'],
    live_update=[
        sync('./build/osm-controller', '/osm-controller'),
      ]
)

docker_build(
    'openservicemesh/osm-crds',
    dockerfile='./dockerfiles/Dockerfile.osm-crds',
    context='./cmd/osm-bootstrap/crds/',
    platform = 'linux/amd64'
)

docker_build_with_restart(
    'openservicemesh/osm-injector',
    context='./build/',
    target='tilt',
    dockerfile_contents = dockerfile_gen('osm-injector'),
    only=[
        'osm-injector'
    ],
    entrypoint=['/osm-injector'],
    live_update=[
        sync('./build/osm-injector', '/osm-injector'),
      ]
)

docker_build_with_restart(
    'openservicemesh/osm-bootstrap',
    context='./build/',
    target='tilt',
    dockerfile_contents = dockerfile_gen('osm-bootstrap'),
    only=[
        'osm-bootstrap'
    ],
    entrypoint=['/osm-bootstrap'],
    live_update=[
        sync('./build/osm-bootstrap', '/osm-bootstrap'),
      ]
)

# https://docs.tilt.dev/tiltfile_config.html#grouping-services-in-web-ui
k8s_resource(workload = "osm-controller", labels = ["osm"])
k8s_resource(workload = "osm-bootstrap", labels = ["osm"])
k8s_resource(workload = "osm-injector", labels = ["osm"])

# build and deploy the demo images on demand
# you can make changes to the demo and redeploy with this resource
local_resource(
    name= "demo",
    cmd = "./scripts/deploy-demo-tilt.sh",
    auto_init = False,
    trigger_mode = TRIGGER_MODE_MANUAL,
    labels = ["osm-demo"]
)
