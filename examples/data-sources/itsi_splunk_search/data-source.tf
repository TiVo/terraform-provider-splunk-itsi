data "itsi_splunk_search" "test_search" {

  search {
    query = <<-EOT
       | makeresults count=5 | eval test = "A"
    EOT

    earliest_time = "-10m"
  }

  search {
    query = <<-EOT
      | makeresults count=5 | eval test = "B"
    EOT

    earliest_time = "-10m"
  }

}
