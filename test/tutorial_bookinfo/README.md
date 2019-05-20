
### Usage
#### Local testing
- make sure that you build and run the tests with the same `BUILD_ID`
- if you want to run glooshot as a local process, set `RUN_GLOOSHOT_LOCAL=1` when you run the test
```bash
export BUILD_ID=tute2e6
export RUN_GLOOSHOT_LOCAL=1
make render-yaml docker-push manifest -B
ginkgo -v .
```
- Note: Local testing is flaky since prometheus pods restart when the config is reloaded.
This interrupts the port-forward
