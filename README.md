# Swing

Swing is a command line tool used to manage upload and download versioned files from AWS S3. It has been thought as an alternative to Git LFS.

Each time a file is uploaded its path and version are added to a CSV, the Swing file, so that it can be commited to VCS to ease tracking.

Files can be easily downloaded by running Swing in the same directory of the Swing file, all modified files will be downloaded using the specified version, if the specified version is not found the latest version will be downloaded instead.

# Configuration

Swing will use the credentials found in the AWS shared credentials file, you can specify which profile to use by setting the env var `AWS_PROFILE`, otherwise the `default` profile will be used. The credentials file is stored in `~/.aws/credentials` on Linux and OS X, and `%UserProfile%\.aws\credentials` on Windows

If no credentials file is found Swing will search for this env vars:
* AWS_ACCESS_KEY_ID
* AWS_SECRET_ACCESS_KEY
* AWS_SESSION_TOKEN (optional)

If neither credentials file and env vars are found Swing will fail.

To create your access keys see the [official AWS documentation][aws-credentials-docs].
To know more about the environment variables see [this other documentation][aws-env-vars-docs].

# Usage

Swing accepts these arguments:

  * `-h` or `--help` prints the help
  * `-f` or `--files` specifies which files to upload
  * `-r` or `--region` AWS region in which the bucket is stored
  * `-b` or `--bucket` name of the bucket to use
  * `-s` or `--swing-file` specifies a custom CSV file used to write files list on upload and read them on download, defaults to `swing.csv`
  * `-d` or `--download` starts download of files found in specified swing file

To upload:

```
swing --files <path_to_file> --bucket <s3_bucket> --region <aws_region>
```

Multiple files can be uploaded at the same time by specifying `--files` each time:

```
--files <path_to_file> --files <path_to_another_file>
```

If the `--region` flag is omitted the region specified in the AWS shared config file for the current profile will be used, if no config file is found Swing will fail. The config file is stored in `~/.aws/config` on Linux and OS X, and in `%UserProfile%\.aws\config` on Windows.

Note that the list of paths saved will be relative to the Swing file and they **MUST** be on the same or below folder.


To download:

```
swing --download
```

If no default Swing file is found in the current folder nothing will be done.


Both on upload and download you can specify a custom Swing file with `--swing-file` and the relative path.

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
