# ecs-export

Get instance hosts and ports for all ECS containers and exports it using go templates

`ecs-export`

```
Usage of ecs-export:
  -c string
    	AWS Cluster arn
  -ep string
    	Regex that should match to a container name(separated by comma)
  -r string
    	AWS Region
  -t string
    	Path to template
```

`ecs-export -c=arn:aws:ecs:eu-west-1:1231313123:cluster/abcd -r=eu-west-1 -ep=container_app(1|2)\_pr -t=example-template`

template-example

```
frontend app
  bind 0.0.0.0:8080
{{range $x := $.Entries}}
  acl {{$x.Name}} hdr(host) -i {{$x.Name}}.domain.com
  use_backend {{$x.Name}} if {{$x.Name}}
{{end}}
{{range $x := $.Entries}}
backend {{$x.Name}}
  server s1 {{$x.Host}}:{{$x.HostPort}}
{{end}}
```

result

```
frontend app
  bind 0.0.0.0:8080

  acl container_app1_pr hdr(host) -i container_app1.domain.com
  use_backend container_app1_pr if container_app1_pr

  acl container_app2_pr hdr(host) -i container_app2_pr.domain.com
  use_backend container_app2_pr if container_app2_pr

backend container_app1_pr
  server s1 10.10.81.255:33089

backend container_app2_pr
  server s1 10.10.81.255:33089
```
