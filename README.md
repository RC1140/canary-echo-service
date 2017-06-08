### Getting started 

If you have a go environment setup you can run run :

`go install gitlab.com/rc1140/canary-echo-backend`

Alternatively download a binary from the releases page.

Ensure the binary downloaded/installed is in your path and then
run 

`canary-echo`

This will start the service in the foreground on :8011

The default username is Admin and the default pass is AdminPass, change 
these via the config file as soon as possible.

Once the service is running open the application and enter your details
into the app. 

Done, thats all there is to setup , simple use the webservice URL as the callback URL for your canary tokens and you will get notified anytime one of them is triggered via a push notification.

### Dependancies

Simply run 

`go get ./...`

to install the required dependancies
