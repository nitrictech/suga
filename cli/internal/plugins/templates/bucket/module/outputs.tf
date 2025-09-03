output "suga" {
    value = {
        id = aws_s3_bucket.bucket.arn
        domain_name = aws_s3_bucket.bucket.bucket_regional_domain_name
        exports = {
            env = {}
            # Export resources to be read into other modules
            resources = {}
        }
    }
}