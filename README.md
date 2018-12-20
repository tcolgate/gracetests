# Testing graceful startup/shutdown in kubernetes

We make extensive use of autoscaling kubernetes clusters. Applications are
scaled with horizontal Pod autoscaling, and the underlying nodes are scaled via
the cluster-autoscaler.

In addition we also make use of AWS Spot instances and GKE preempible instances,
due to the substantial cost saving they can provide.

Whilst cluster scale up is generally seamless, (assuming we scale up
sufficiently in advance of the need for the extra capacity), cluster scale down
causes small, but noticable, interruption to services.

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

## Gotchas

### Docker ENTRYPOINT

### Correct signal (SIGTERM)

### Correct signal (SIGTERM)






