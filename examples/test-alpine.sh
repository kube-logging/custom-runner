#!/usr/bin/env bash

# build local image
docker build -f Dockerfile.alpine -t alpine-cr:test .

# load image to kind
kind load docker-image alpine-cr:test

# create secret
kubectl apply -f- <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: default
  labels:
type: Opaque
data:
  testdata: $(echo foobar | base64)
EOF

# create pod, mount up secret
#   - "{\"events\":{\"onStart\":[{\"exec\":{\"key\":\"foobar\", \"command\":\"echo foo\"}}]}}"
#   - '{"events":{"onStart":[{"exec":{"key":"foobar", "command":"echo foo"}}]}}'
kubectl apply -f- <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - image: alpine-cr:test
    name: alpine-cr-test
    command: ["/runner"]
    args:
      - -debug
      - -cfgjson
      - '
{
  "events": {
    "onStart": [
      {
        "exec": {
          "key": "foobar",
          "command": "echo foo"
        }
      }
    ],
    "onFileCreate": {
      "/test/..data" : [
        {
          "exec": {
            "key": "info",
            "command": "echo secret changed!"
          }  
        },
        {
          "exec": {
              "key": "reload",
              "command": "echo RELOAD | socat - UNIX-CONNECT:/foo/bar"
          }
        }  
      ]
    }
  }
}
      '
    volumeMounts:
    - mountPath: /test
      name: test-volume
  volumes:
  - name: test-volume
    secret:
      secretName: test-secret
    #   items:
    #     - key: testdata
    #       path: mydata
EOF

# test mount
# k exec -it test-pod -- sh
# k exec test-pod -- ls -l /test
# k exec test-pod -- cat /test/testdata

