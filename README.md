# Testing graceful startup/shutdown in kubernetes

We make extensive use of autoscaling kubernetes clusters. Applications are
scaled with horizontal Pod autoscaling, and the underlying nodes are scaled via
the cluster-autoscaler.

In addition we also make use of AWS Spot instances and GKE preempible instances,
due to the substantial cost saving they can provide.

Whilst cluster scale up is generally seamless, (assuming we scale up
sufficiently in advance of the need for the extra capacity), cluster scale down
causes small, but noticable, interruption to services.

There are also several causes of Pod termination.

Some are caused by kubernetes directly:
- Pod terminated manually by administrator.
- HPA scales down, terminates the pod.
- Rolling deployment, or deployment scales, terminates the pod.
- Cluster autoscaler scales down the node instance group. Evicting
  remaining pods on the node.

In these instances a pod is terminated via the kube API. We can reasonably
expect that any time the API deletes a pod, it should be done following the
full graceful termination procedure.

There are two other common situations.
- The cloud provider terminates a Spot or Preemptible instance.
- A cloud administrator terminates a node.

In these circumstances the shutdown is instigated direct to the node, out of
band from the kubernetes API. It is the node's responsibility to properly shut
down. On receipt of the shutdown request the node should drain itself.
With our Kops AWS deployment some additional configuration is needed at this
time.  (https://medium.com/skedulo-engineering/minimising-the-impact-of-kubernetes-node-shutdowns-11cd8233e2fb)

## Hypothesis:

Simply settings terminationGracePeriodSeconds on a Pod/Deployment is
insufficient to ensure gracedul scale down. Services must cooperate in the 
shutdown process, and require non-trival processes to do so.

To smoothly deploy, and gracefully scale, we must fully implement graceful
shutdown.

1. We must listen for SIGTERM (many examples assume SIGINT, e.g. Go
   os.Interrupt)
2. We must continue to receieve and process queries for a time after SIGTERM.
3. We should fail our healthcheck to clearly indicate we are nolong ready. We
   should do this for sufficiently long to full fail our readiness probe.
4. We should shutdown once we have processed our inflight requests

## Testing

- deploy a trivial service
- fix the number of pods (to e.g. 2 pods)
- run a continious load (e.g. 1000 req/s) for a reasonable duration
  long enough to allow two or three complete redeploys
- edit the deployment, update an attribute on the pods, to force a redeploy

Two version of the services will be deployed. One which is actively involved
in graceful shutdown, fails probes, continues to acccept requests, and 
gracefully shuts down (processing in-flight).

The second version, ungracefully quits.

Once basic graceful termination is confirmed further test is needed. We need
to verify that graceful node termination correct respects graceful pod
termination.

- Force a deployment to a group of nodes.
- run a continious load for a reasonable duration
- Terminate one of the running nodes. 

## Gotchas

### Node termination

Our existing kops clusters were not respecting ACPI shutdown signals. Further
configuration is required.

### Docker ENTRYPOINT

If entrypoint is set using a string rather than an array, a shell will be
interjected into the docker container. This shell will be PID 1, not the
requested command. In linux pid 1 processes will ignore SIGTERM and SIGINT
by default, and will not propagate it to children.

Since we never get SIGTERM we do not know we are shutting down. Kubernetes will
kill us eventually, but we may need to ensure we have processed any slow,
in-flight, requests first.

The end result is that kubernetes will SIGKILL the container after it fails to
exit during the grace period.

This accidental hard exit can have the desired results, as long as all queries
have completed by the time the pod is forcibly destroyed.


### Correct signal (SIGTERM)

Many on-line example , (particulary for Go), demonstrate graceful shutdown using
SIGINT (sent, for example, via ctrl-c at a terminal). This is not sufficient in
kubernetes, where a SIGTERM is sent (as is more normal for services, rather than
CLI tools). In Go, we need to listen for syscall.SIGTERM in addition to
os.Interrupt (if we wish to be able to test graceful shutdown with ctrl-c). In
python sani at leasts respects both.


## Additional considerations

- Health checks from a cloud provided ingress may not follow the cadence of the
  specified health checks.
- externalTrafficPolicy further confuses matters.

## Findings

- Node shutdown should properly drain a node via the API.
- Requests arrive after the arrival of the termination signal. Some period of
  listening after the arrival of the signal is essential.
- Consequently, the biggest risk is exiting too quickly.
- We don't have to actually fail the healtcheck, but doing so does not cause any
  harm.
  - This may not be the case for Type: Loadbalancer services where a cloud LB is
    actively healthchecking the service directly. In these cases, the suspicion
    is that actively failing the health check help in updating external LBs. It
    is also important to ensure that your load balancer respect graceful
    shutdown of connections (e.g. using the
    service.beta.kubernetes.io/aws-load-balancer-connection-draining-enabled
    annotations)
- actively stopping listening is not stricty neccesary, but this is built in
  to the shutdown procedures of many http servers (Go and Python/sanic at
  least). These shutdown processes are important for ensuring in-flight requests
  are complete.

