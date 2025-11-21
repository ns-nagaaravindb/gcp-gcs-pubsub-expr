Terraform Module
1. create bucket called and create pubsub notification for when there file created in it 
2. gcs bucket (test-dp-gcspubsub-bucket) and pubsub topic name (test-dp-gcspubsub-bucket)
3. it should use glcoud login credentials 
4. terraform production grade standard /compliance should be used. 
5. Should have one Readme.md for each sub module. should not create too much documentation 

Application :
1. Simple go app to create files in bucket and read from pubsub topic to verify the we received file creation events from pubsub notification and derive the file URL download and print the content of this file .
2. production grade standard /compliance should be used. 
3. Should have one Readme.md for each sub module. should not create too much documentation 

make file for demo/testing :
1.  make file for create and delete infra (gcs,pubsub notification, pubsub) 
2.  create file and reading data from pubsub for notification & print the file contents
3.  Misc  