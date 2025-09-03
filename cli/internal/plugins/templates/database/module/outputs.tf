output "suga" {
    value = {
        id          = ""
        exports = {
            # Export known service outputs
            services = {
                # This will normally be mapped as part of the supplied services
                # in the suga input variable
                # my-service-name = {
                #     env = {}
                # }
            }
            resources = {}
        }
    } 
}
