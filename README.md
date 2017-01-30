# dyndns53

dyndns53 can be used to run your own [dynamic DNS][] service with [Amazon Route
53][] by updating a domain of your choice with the public IP address of the
machine it runs on.

[dynamic DNS]: https://en.wikipedia.org/wiki/Dynamic_DNS
[Amazon Route 53]: https://aws.amazon.com/route53/

## Installation

[Install Go][] and issue:

    $ go get github.com/agorf/dyndns53

[Install Go]: https://golang.org/doc/install

## Configuration

Log in to the [AWS management console][] and follow the steps below. You need to
do this _once_.

[AWS management console]: https://console.aws.amazon.com/

### Step 1: Create a hosted zone

Go to the [Route 53 Hosted Zones page][] and click _Create Hosted Zone_.

Fill in your domain name in _Domain Name_ and choose _Public Hosted Zone_ for
_Type_, then click _Create_.

In the hosted zone page, click _Back to Hosted Zones_ and note down the _Hosted
Zone ID_ since you will need it in the next step.

[Route 53 Hosted Zones page]: https://console.aws.amazon.com/route53/home#hosted-zones:

### Step 2: Create an IAM policy

Go to to the [IAM Policies page][] and click _Create Policy_.

Click _Select_ on the _Policy Generator_ section.

In the following form, choose _Allow_ for _Effect_, _Amazon Route 53_ for _AWS
Service_, _ChangeResourceRecordSets_ for _Actions_, fill in
`arn:aws:route53:::hostedzone/<HOSTEDZONEID>` for _Amazon Resource Name (ARN)_
(replacing `<HOSTEDZONEID>` with the hosted zone id from the previous step) and
click _Add Statement_ and then _Next Step_.

Fill in a name for _Policy Name_ so that you can look up the policy later and
click _Create Policy_.

[IAM Policies page]: https://console.aws.amazon.com/iam/home#/policies

### Step 3: Create an IAM user

Go to the [IAM Users page][] and click the _Add user_ button.

Fill in the user name, check _Programmatic access_ for _Access type_ and click
_Next: Permissions_.

On the permissions page, click the last option _Attach existing policies
directly_ and fill in the _Search_ field with the name of the policy you created
in the previous step. Click on the policy to check it and click _Next: Review_.

Click _Create user_ and you will be presented with an _Access key ID_ and a
_Secret access key_, the credentials dyndns53 needs to access the service
programmatically. Don't close the window since you will need them in the next
step.

[IAM Users page]: https://console.aws.amazon.com/iam/home#/users

### Step 4: Create an AWS credentials file

In the machine you plan to run dyndns53, issue:

    $ mkdir -p ~/.aws
    $ touch ~/.aws/credentials
    $ chmod go-r ~/.aws/credentials # prevent other users from reading the file

Edit the file with your editor:

    $ vim ~/.aws/credentials

And write:

    [dyndns53]
    aws_access_key_id = <ACCESS_KEY_ID>
    aws_secret_access_key = <SECRET_ACCESS_KEY>

Replacing `<ACCESS_KEY_ID>` and `<SECRET_ACCESS_KEY>` with the actual values you
were presented in the previous step.

You are now ready to use dyndns53.

## Usage

Running the program with no arguments prints a usage text:

    $ dyndns53
    Usage of dyndns53:
      -log string
            file name to log to (default is stdout)
      -name string
            record set name; must end with "."
      -ttl int
            TTL (time to live) in seconds (default 300)
      -type string
            record set type; "A" or "AAAA" (default "A")
      -zone string
            hosted zone id

You can set [Cron][] (with `crontab -e`) to run the program e.g. every five
minutes:

    */5 * * * * dyndns53 -name mydomain.com. -zone HOSTEDZONEID

Where `mydomain.com.` is the name of the record set you want to update (note
that it must end with a `.`) and `HOSTEDZONEID` is the id of the [hosted zone][]
the record set belongs to.

If the record set does not exist, it will be created the first time dyndns53
runs.

[Cron]: https://en.wikipedia.org/wiki/Cron
[hosted zone]: http://docs.aws.amazon.com/Route53/latest/DeveloperGuide/ListInfoOnHostedZone.html

## License

Licensed under the MIT license (see [LICENSE.txt][]).

[LICENSE.txt]: https://github.com/agorf/dyndns53/blob/master/LICENSE.txt

## Author

Angelos Orfanakos, http://agorf.gr/
