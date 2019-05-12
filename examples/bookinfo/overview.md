
Hello, here is a demo of how to use Glooshot define and execute chaos engineering experiments.

Glooshot understands how to interface - and interfere with - the properties of a service mesh. With Glooshot, you can specify faults and delays between any services hosted in a service mesh. These faults will exist until any of the stop conditions you specify are met.

I will show you some simple examples to get you started.

Let's look at a simple app running in a service mesh.

Here is a bookinfo app from the Istio sample app repo. It is a simple app that shows reviews and ratings of books. The app consists of the "product" page and two supporting services that provide reviews and ratings.

The information flow is like this:
- The product service asks the reviews service for data
  - The reviews service asks the ratings service for data.
    - The ratings service responds
  - The reviews service responds
- The product page renders the data, as you see here

Web Page <-> Product Service <-> Reviews Service <-> Ratings Service

One concern you may have with an app like this is what happens to the product page if there is a failure in the Ratings Service? Ideally, the Reviews Service will handle the failure gracefully, and return what data it can.

For demonstration purposes, we have introduced a "vulnerable" version of the Reviews Service. In this vulnerable version, if the Ratings Service is unavailable, the Reviews Service will fail.

We will use Glooshot to find this weakness. When it fails, we will replace the service with a more robust version and show that the failure mode is less severe.

First, let's apply a failure to the reviews service itself, to see what the worst case failure looks like.

I will use supergloo to create a fault.

For simplicity, I've written a helper script the long form of all these commands in the file `demo.sh`. So that you don't have to watch me type, I will call the commands by id. The script prints out the corresponding commands and applies them. Review the file for more details. To get started, try `demo.sh help-verbose`.

Ok let's get started. Here is a demonstration of the "worst-case" failure mode:
```
./demo.sh 6
```

Notice that there is no information returned from the Reviews Service or the Ratings Service.

Let's delete that failure and refresh.
```
./demo.sh 7
```

Ok, back to normal.

Let's apply a Glooshot experiment that will run until a timeout is reached. Let's say 30 seconds.

I will now apply the experiment.
```
./demo.sh 14
```

Refresh the page, and see that the experiment is active. The Reviews Service has failed completely.

After 30 seconds, the experiment will end, let's refresh and see.

Ok, back to normal. Let's inspect the experiment results.
```
./demo.sh 15
```

We see that the experiment succeeded. That is, that the experiment ended by timing out, rather than by exceeding a metric threshold.

In a real experiment, we want to make sure that our system is not taking too much damage. We want to terminate the experiment when we reach our critical threshold values.

Let's look at prometheus and choose some good threshold values.

For our app, let's set a limit on the number of Server Error (500 code) responses recieved by the product service. Looking at prometheus, we have so far seen X of these errors. Let's set our threshold to X+10. (Note, more advanced configurations are possible, but we'll keep it simple for now).

Ok, let's specify an experiment with this stop condition. We'll increase the timeout to 100 hours just to make sure that the stop condition is the metric threshold.

Apply the experiment.
```
./demo.sh 16
```

Refresh the page and notice that the failure has returned.
Let's refresh the page a few times to trigger the stop condition.

We have now exceeded our limit and the experiment should have ended. Let's reload the page and confirm that we see the reviews service data again.

Cool, the experiment ended as expected.

Let's inspect the results data.
```
./demo.sh 17
```

We see that the experiment was terminated when a threshold was exceeded.

Great, now let's deploy a more robust reviews service and see if this failure is cleared up.
```
./demo.sh 9
```

We can do that with supergloo.

Reload the page, ok, looks the same. Now let's apply the experiment again and see if this new version is more tolerant to failure.

Apply the experiment with Glooshot

Reload the page

Note that the Reviews data is returned, and only the ratings data is missing. Great,  we have made our app more fault tolerant!
