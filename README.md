### Getting started 

If you have a go environment setup you can run run :

`go install gitlab.com/rc1140/canary-echo-backend`

Alternatively download a binary from the releases page.

Ensure the binary downloaded/installed is in your path and then
run 

`canary-echo`

This will start the service in the foreground on :8011

Copy the sample config to echo-config.yaml

Set the value for all the keys provided in the sample config.

Once the service is running open the mobile application and enter your details
into the app and using your hostname in the host field. 

Done, thats all there is to setup. You can now simply use the webservice URL as the callback URL for your canary tokens and you will get notified anytime one of them is triggered via a push notification.

### Dependancies

Simply run 

`go get ./...`

to install the required dependancies
