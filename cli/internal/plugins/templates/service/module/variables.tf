# There are mandatory variables that will be provided by the Suga deployment engine
# to this module.
variable "suga" {
  type = object({
    name       = string
    stack_id   = string
    image_id   = string
    schedules  = optional(map(object({
      cron_expression = string
      path            = string
    })), {})
    env        = map(string)
    identities = map(object({
      exports = map(string)
    }))
  })
}

# You may include any additional variables you wish to configure your service here.
