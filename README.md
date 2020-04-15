# Swing

Swing is a command line tool used to manage upload and download versioned files from AWS S3. It has been thought as an alternative to Git LFS.

Versioning **MUST** be enabled on the buckets used with Swing for it to work correctly.

Each time a file is uploaded its path and version are added to a CSV, the Swing file, so that it can be commited to VCS to ease tracking.

Files can be easily downloaded by running Swing in the same directory of the Swing file, all modified files will be downloaded using the specified version, if the specified version is not found the latest version will be downloaded instead.

# Configuration

Swing will use the credentials found in the AWS shared credentials file, you can specify which profile to use by setting the env var `AWS_PROFILE`, otherwise the `default` profile will be used. The credentials file is stored in `~/.aws/credentials` on Linux and OS X, and `%UserProfile%\.aws\credentials` on Windows

If no credentials file is found Swing will search for this env vars:
* AWS_ACCESS_KEY_ID
* AWS_SECRET_ACCESS_KEY
* AWS_SESSION_TOKEN (optional)

If the user's account has more a MFA device associated it will be prompted for the token generated.
In case the devices are more then one it will be first prompted to pick a device to use for MFA.

If neither credentials file and env vars are found Swing will fail.

To create your access keys see the [official AWS documentation][aws-credentials-docs].
To know more about the environment variables see [this other documentation][aws-env-vars-docs].

# Necessary AWS IAM permissions

S3 Buckets permissions 

```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:ListAllMyBuckets",
                "s3:GetBucketLocation",
                "s3:GetBucketAcl"
            ],
            "Resource": "arn:aws:s3:::*"
        },
        {
            "Sid": "devBucketsAccess",
            "Effect": "Allow",
            "Action": [
                "s3:*"
            ],
            "Resource": [
                "arn:aws:s3:::mybucket*"
            ]
        }
    ]
}
```

IAM permissions 

```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "GetUserInformations",
            "Effect": "Allow",
            "Action": [
                "iam:GetUser",
                "iam:ListMFADevices"
            ],
            "Resource": "arn:aws:iam::*:user/${aws:username}"
        }
    ]
}
```
# Usage

Swing accepts these arguments:

  * `-h` or `-help` prints the help
  * `-r` or `-region` AWS region in which the bucket is stored
  * `-b` or `-bucket` name of the bucket to use
  * `-s` or `-swing-file` specifies a custom CSV file used to write files list on upload and read them on download, defaults to `swing.csv`
  * `-d` or `-download` starts download of files found in specified swing file
  * `-h` or `-help` shows the help text

To upload:

```
swing --bucket <s3_bucket> --region <aws_region> <path_to_file>
```

Multiple files can be uploaded at the same time if necessary:

```
swing --bucket <s3_bucket> --region <aws_region> <path_to_file_1> <path_to_file_2> <path_to_file_3>
```

If the `--region` flag is omitted the region specified in the AWS shared config file for the current profile will be used, if no config file is found Swing will fail. The config file is stored in `~/.aws/config` on Linux and OS X, and in `%UserProfile%\.aws\config` on Windows.

Note that the list of paths saved will be relative to the Swing file and they **MUST** be on the same or below folder.


To download:

```
swing --download
```

If no default Swing file is found in the current folder nothing will be done.


Both on upload and download you can specify a custom Swing file with `--swing-file` and the relative path.

# Swing file

The Swing file is a CSV not meant to be edited manually, each field is separated by a semicolon (`;`).

This is an example:

```
file;region;bucket;md5;version_id
data/my_database.sqlite;eu-central-1;test.bucket.com;412300cb44e55e67dced78c42e7fbcaa;Edhf5InUn20iG8errAxTo3qjZx.OCXjE
big_binary;eu-central-1;test.bucket.com;ac6f71a29304799218cc2427f567f436;paZmQA4Di4kHbjbY1623l1raqqgYWRG3
libraries/other_lib.a;eu-central-1;test.bucket.com;f572bf8f0ca53b342aca927a509d3f6c;8_QHfD.C050sVkQKtqEz1jay7ZqGn2lZ
```

`file` is the path of the file relative to the Swing file.
`region` and `bucket` are respectively the AWS region in which the bucket is hosting the file.
`md5` is the hash of the file calculated during the upload.
`version_id` is the id of the object uploaded version, returned by AWS after successful upload.

# Build

Go 1.14 or later are required.

To build run:

```
go build
```

Or just install it with:

```
go install
```

[aws-env-vars-docs]: https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html
[aws-credentials-docs]: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html#Using_CreateAccessKey
