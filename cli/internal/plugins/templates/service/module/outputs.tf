# These are mandatory outputs that will be consumed by the Suga deployment engine
output "suga" {
  value = {
    id          = aws_lambda_function.function.arn
    domain_name = split("/", aws_lambda_function_url.endpoint.function_url)[2]
    exports = {
      resources = {
        # Output any resource identifiers that need to be referenced by other services here.
        # By convention names that can be used to retrieve these values via a terraform data resource should be used
        # See: for an example
      }
    }
  }
}

