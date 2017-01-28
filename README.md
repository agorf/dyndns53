# dyndns53

dyndns53 can be used to set up your own [dynamic DNS][] service. It does that by
accessing the [Amazon Route 53][Route 53 API] ([DNS][]) service, updating a
domain of your choice with the public IP address of the machine it runs on.

[dynamic DNS]: https://en.wikipedia.org/wiki/Dynamic_DNS
[DNS]: https://www.expeditedssl.com/aws-in-plain-english

## Installation

[Install Go][], if you haven't, and issue:

    $ go get github.com/agorf/dyndns53

[Install Go]: https://golang.org/doc/install

## Configuration

Necessary options are passed from the command-line when running the program. The
only thing you need to set up in terms of configuration is an [AWS
credentials][] file to access the [Route 53 API][]:

    $ mkdir -p ~/.aws
    $ touch ~/.aws/credentials
    $ chmod go-r ~/.aws/credentials # prevent other users from reading the file

Edit the file with your editor:

    $ vim ~/.aws/credentials

The file should contain:

    [dyndns53]
    aws_access_key_id = <ACCESS_KEY_ID>
    aws_secret_access_key = <SECRET_ACCESS_KEY>

Where `<ACCESS_KEY_ID>` and `<SECRET_ACCESS_KEY>` are the actual values from the
[AWS IAM service][IAM].

[AWS credentials]: https://aws.amazon.com/blogs/security/a-new-and-standardized-way-to-manage-credentials-in-the-aws-sdks/
[Route 53 API]: http://docs.aws.amazon.com/Route53/latest/APIReference/API_ChangeResourceRecordSets.html
[IAM]: http://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html#Using_CreateAccessKey

## Usage

Running the program with no arguments prints a usage text:

    $ dyndns53
    Usage of dyndns53:
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

[Cron]: https://en.wikipedia.org/wiki/Cron
[hosted zone]: http://docs.aws.amazon.com/Route53/latest/DeveloperGuide/ListInfoOnHostedZone.html

## License

Licensed under the MIT license (see [LICENSE.txt][]).

[LICENSE.txt]: https://github.com/agorf/dyndns53/blob/master/LICENSE.txt

## Author

Angelos Orfanakos, http://agorf.gr/
