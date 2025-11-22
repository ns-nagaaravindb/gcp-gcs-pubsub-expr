.PHONY: help terraform-init terraform-plan terraform-apply terraform-destroy terraform-output \
	app-build app-create-file app-listen app-test clean \
	check-deps check-gcloud check-terraform check-go

# Configuration
PROJECT_ID ?= $(shell gcloud config get-value project 2>/dev/null)
CONSUMER_PROJECT_ID ?= compute-k8s-qe
BUCKET_NAME ?= test-dp-gcspubsub-bucket
TOPIC_NAME ?= test-dp-gcspubsub-bucket
REGION ?= us-central1
TERRAFORM_DIR = pubsub-notification-terraform
APP_DIR = app
APP_BINARY = gcs-pubsub-app
CREATE_CROSS_PROJECT ?= true

# Colors for output
GREEN = \033[0;32m
YELLOW = \033[0;33m
RED = \033[0;31m
NC = \033[0m # No Color

help: ## Show this help message
	@echo "$(GREEN)Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'

check-deps: check-gcloud check-terraform check-go ## Check all dependencies

check-gcloud: ## Check if gcloud is installed and authenticated
	@which gcloud > /dev/null || (echo "$(RED)Error: gcloud not found. Please install Google Cloud SDK$(NC)" && exit 1)
	@gcloud auth application-default print-access-token > /dev/null 2>&1 || (echo "$(RED)Error: Not authenticated. Run: gcloud auth application-default login$(NC)" && exit 1)
	@echo "$(GREEN)✓ gcloud is installed and authenticated$(NC)"

check-terraform: ## Check if terraform is installed
	@which terraform > /dev/null || (echo "$(RED)Error: terraform not found. Please install Terraform$(NC)" && exit 1)
	@echo "$(GREEN)✓ terraform is installed$(NC)"

check-go: ## Check if go is installed
	@which go > /dev/null || (echo "$(RED)Error: go not found. Please install Go$(NC)" && exit 1)
	@echo "$(GREEN)✓ go is installed$(NC)"

# Terraform targets
terraform-init: check-terraform ## Initialize Terraform
	@echo "$(GREEN)Initializing Terraform...$(NC)"
	cd $(TERRAFORM_DIR) && terraform init

terraform-plan: terraform-init ## Plan Terraform changes
	@if [ -z "$(PROJECT_ID)" ]; then \
		echo "$(RED)Error: PROJECT_ID not set. Set it via: make terraform-plan PROJECT_ID=your-project-id$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Planning Terraform changes...$(NC)"
	@if [ -n "$(CONSUMER_PROJECT_ID)" ] && [ "$(CREATE_CROSS_PROJECT)" = "true" ]; then \
		echo "$(YELLOW)Cross-project mode: Consumer project $(CONSUMER_PROJECT_ID)$(NC)"; \
		cd $(TERRAFORM_DIR) && terraform plan \
			-var="project_id=$(PROJECT_ID)" \
			-var="region=$(REGION)" \
			-var="bucket_name=$(BUCKET_NAME)" \
			-var="topic_name=$(TOPIC_NAME)" \
			-var="consumer_project_id=$(CONSUMER_PROJECT_ID)" \
			-var="create_cross_project_subscription=true"; \
	else \
		cd $(TERRAFORM_DIR) && terraform plan \
			-var="project_id=$(PROJECT_ID)" \
			-var="region=$(REGION)" \
			-var="bucket_name=$(BUCKET_NAME)" \
			-var="topic_name=$(TOPIC_NAME)"; \
	fi

terraform-apply: terraform-init ## Apply Terraform configuration
	@if [ -z "$(PROJECT_ID)" ]; then \
		echo "$(RED)Error: PROJECT_ID not set. Set it via: make terraform-apply PROJECT_ID=your-project-id$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Applying Terraform configuration...$(NC)"
	@if [ -n "$(CONSUMER_PROJECT_ID)" ] && [ "$(CREATE_CROSS_PROJECT)" = "true" ]; then \
		echo "$(YELLOW)Cross-project mode: Consumer project $(CONSUMER_PROJECT_ID)$(NC)"; \
		cd $(TERRAFORM_DIR) && terraform apply \
			-var="project_id=$(PROJECT_ID)" \
			-var="region=$(REGION)" \
			-var="bucket_name=$(BUCKET_NAME)" \
			-var="topic_name=$(TOPIC_NAME)" \
			-var="consumer_project_id=$(CONSUMER_PROJECT_ID)" \
			-var="create_cross_project_subscription=true" \
			-auto-approve; \
	else \
		cd $(TERRAFORM_DIR) && terraform apply \
			-var="project_id=$(PROJECT_ID)" \
			-var="region=$(REGION)" \
			-var="bucket_name=$(BUCKET_NAME)" \
			-var="topic_name=$(TOPIC_NAME)" \
			-auto-approve; \
	fi
	@echo "$(GREEN)✓ Infrastructure created successfully$(NC)"

terraform-destroy: terraform-init ## Destroy Terraform infrastructure
	@if [ -z "$(PROJECT_ID)" ]; then \
		echo "$(RED)Error: PROJECT_ID not set. Set it via: make terraform-destroy PROJECT_ID=your-project-id$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Destroying Terraform infrastructure...$(NC)"
	@if [ -n "$(CONSUMER_PROJECT_ID)" ] && [ "$(CREATE_CROSS_PROJECT)" = "true" ]; then \
		cd $(TERRAFORM_DIR) && terraform destroy \
			-var="project_id=$(PROJECT_ID)" \
			-var="region=$(REGION)" \
			-var="bucket_name=$(BUCKET_NAME)" \
			-var="topic_name=$(TOPIC_NAME)" \
			-var="consumer_project_id=$(CONSUMER_PROJECT_ID)" \
			-var="create_cross_project_subscription=true" \
			-auto-approve; \
	else \
		cd $(TERRAFORM_DIR) && terraform destroy \
			-var="project_id=$(PROJECT_ID)" \
			-var="region=$(REGION)" \
			-var="bucket_name=$(BUCKET_NAME)" \
			-var="topic_name=$(TOPIC_NAME)" \
			-auto-approve; \
	fi
	@echo "$(GREEN)✓ Infrastructure destroyed$(NC)"

terraform-output: terraform-init ## Show Terraform outputs
	@cd $(TERRAFORM_DIR) && terraform output

# Application targets
app-build: check-go ## Build the Go application
	@echo "$(GREEN)Building application...$(NC)"
	cd $(APP_DIR) && go build -o ../$(APP_BINARY) .
	@echo "$(GREEN)✓ Application built: $(APP_BINARY)$(NC)"

app-create-file: app-build ## Create a test file in the bucket
	@if [ -z "$(PROJECT_ID)" ]; then \
		echo "$(RED)Error: PROJECT_ID not set. Set it via: make app-create-file PROJECT_ID=your-project-id$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Creating test file...$(NC)"
	./$(APP_BINARY) \
		-mode=create \
		-project=$(PROJECT_ID) \
		-bucket=$(BUCKET_NAME) \
		-topic=$(TOPIC_NAME) \
		-file=test-$(shell date +%s).txt \
		-content="Test file created at $(shell date)"

app-listen: app-build ## Listen for Pub/Sub notifications
	@if [ -z "$(PROJECT_ID)" ]; then \
		echo "$(RED)Error: PROJECT_ID not set. Set it via: make app-listen PROJECT_ID=your-project-id$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Listening for notifications (Press Ctrl+C to stop)...$(NC)"
	@if [ -n "$(CONSUMER_PROJECT_ID)" ]; then \
		echo "$(YELLOW)Consumer project: $(CONSUMER_PROJECT_ID)$(NC)"; \
		./$(APP_BINARY) \
			-mode=listen \
			-project=$(PROJECT_ID) \
			-consumer-project=$(CONSUMER_PROJECT_ID) \
			-bucket=$(BUCKET_NAME) \
			-topic=$(TOPIC_NAME); \
	else \
		./$(APP_BINARY) \
			-mode=listen \
			-project=$(PROJECT_ID) \
			-bucket=$(BUCKET_NAME) \
			-topic=$(TOPIC_NAME); \
	fi

app-test: app-build ## Create file and listen for notification
	@if [ -z "$(PROJECT_ID)" ]; then \
		echo "$(RED)Error: PROJECT_ID not set. Set it via: make app-test PROJECT_ID=your-project-id$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Creating file and listening for notification...$(NC)"
	@if [ -n "$(CONSUMER_PROJECT_ID)" ]; then \
		echo "$(YELLOW)Consumer project: $(CONSUMER_PROJECT_ID)$(NC)"; \
		./$(APP_BINARY) \
			-mode=both \
			-project=$(PROJECT_ID) \
			-consumer-project=$(CONSUMER_PROJECT_ID) \
			-bucket=$(BUCKET_NAME) \
			-topic=$(TOPIC_NAME) \
			-file=test-$(shell date +%s).txt \
			-content="Test file created at $(shell date)"; \
	else \
		./$(APP_BINARY) \
			-mode=both \
			-project=$(PROJECT_ID) \
			-bucket=$(BUCKET_NAME) \
			-topic=$(TOPIC_NAME) \
			-file=test-$(shell date +%s).txt \
			-content="Test file created at $(shell date)"; \
	fi

# Cleanup targets
clean: ## Clean build artifacts
	@echo "$(GREEN)Cleaning build artifacts...$(NC)"
	rm -f $(APP_BINARY)
	cd $(TERRAFORM_DIR) && rm -rf .terraform terraform.tfstate* .terraform.lock.hcl
	@echo "$(GREEN)✓ Cleanup complete$(NC)"

clean-all: terraform-destroy clean ## Destroy infrastructure and clean artifacts
	@echo "$(GREEN)✓ All cleanup complete$(NC)"

# Demo/Testing workflow
demo: check-deps terraform-apply app-test ## Full demo: create infra, create file, and listen
	@echo "$(GREEN)✓ Demo completed$(NC)"

# Misc targets
info: ## Show current configuration
	@echo "$(GREEN)Current Configuration:$(NC)"
	@echo "  PROJECT_ID: $(PROJECT_ID)"
	@echo "  CONSUMER_PROJECT_ID: $(CONSUMER_PROJECT_ID)"
	@echo "  BUCKET_NAME: $(BUCKET_NAME)"
	@echo "  TOPIC_NAME: $(TOPIC_NAME)"
	@echo "  REGION: $(REGION)"
	@echo "  CREATE_CROSS_PROJECT: $(CREATE_CROSS_PROJECT)"

test-infra: terraform-plan ## Test infrastructure configuration without applying
	@echo "$(GREEN)✓ Infrastructure configuration is valid$(NC)"

# Cross-project specific targets
cross-project-apply: ## Apply infrastructure with cross-project consumer
	@if [ -z "$(PROJECT_ID)" ] || [ -z "$(CONSUMER_PROJECT_ID)" ]; then \
		echo "$(RED)Error: Both PROJECT_ID and CONSUMER_PROJECT_ID must be set$(NC)"; \
		echo "Usage: make cross-project-apply PROJECT_ID=publisher-project CONSUMER_PROJECT_ID=consumer-project"; \
		exit 1; \
	fi
	@$(MAKE) terraform-apply CREATE_CROSS_PROJECT=true

cross-project-destroy: ## Destroy infrastructure with cross-project consumer
	@if [ -z "$(PROJECT_ID)" ] || [ -z "$(CONSUMER_PROJECT_ID)" ]; then \
		echo "$(RED)Error: Both PROJECT_ID and CONSUMER_PROJECT_ID must be set$(NC)"; \
		echo "Usage: make cross-project-destroy PROJECT_ID=publisher-project CONSUMER_PROJECT_ID=consumer-project"; \
		exit 1; \
	fi
	@$(MAKE) terraform-destroy CREATE_CROSS_PROJECT=true

cross-project-listen: app-build ## Listen for notifications in consumer project
	@if [ -z "$(PROJECT_ID)" ] || [ -z "$(CONSUMER_PROJECT_ID)" ]; then \
		echo "$(RED)Error: Both PROJECT_ID and CONSUMER_PROJECT_ID must be set$(NC)"; \
		echo "Usage: make cross-project-listen PROJECT_ID=publisher-project CONSUMER_PROJECT_ID=consumer-project"; \
		exit 1; \
	fi
	@echo "$(GREEN)Listening for notifications in consumer project (Press Ctrl+C to stop)...$(NC)"
	GCP_PROJECT_ID=$(PROJECT_ID) GCP_CONSUMER_PROJECT_ID=$(CONSUMER_PROJECT_ID) \
	./$(APP_BINARY) \
		-mode=listen \
		-bucket=$(BUCKET_NAME) \
		-topic=$(TOPIC_NAME)

cross-project-test: app-build ## Full cross-project test: create file in publisher, listen in consumer
	@if [ -z "$(PROJECT_ID)" ] || [ -z "$(CONSUMER_PROJECT_ID)" ]; then \
		echo "$(RED)Error: Both PROJECT_ID and CONSUMER_PROJECT_ID must be set$(NC)"; \
		echo "Usage: make cross-project-test PROJECT_ID=publisher-project CONSUMER_PROJECT_ID=consumer-project"; \
		exit 1; \
	fi
	@echo "$(GREEN)Testing cross-project setup...$(NC)"
	GCP_PROJECT_ID=$(PROJECT_ID) GCP_CONSUMER_PROJECT_ID=$(CONSUMER_PROJECT_ID) \
	./$(APP_BINARY) \
		-mode=both \
		-bucket=$(BUCKET_NAME) \
		-topic=$(TOPIC_NAME) \
		-file=cross-project-test-$(shell date +%s).txt \
		-content="Cross-project test file created at $(shell date)"

cross-project-demo: check-deps cross-project-apply cross-project-test ## Full cross-project demo
	@echo "$(GREEN)✓ Cross-project demo completed$(NC)"

cross-project-verify: ## Verify cross-project infrastructure and test connectivity
	@if [ -z "$(PROJECT_ID)" ] || [ -z "$(CONSUMER_PROJECT_ID)" ]; then \
		echo "$(RED)Error: Both PROJECT_ID and CONSUMER_PROJECT_ID must be set$(NC)"; \
		echo "Usage: make cross-project-verify PROJECT_ID=publisher-project CONSUMER_PROJECT_ID=consumer-project"; \
		exit 1; \
	fi
	@echo "$(GREEN)Running cross-project verification test...$(NC)"
	@cd $(TERRAFORM_DIR) && ./test-cross-project.sh $(PROJECT_ID) $(CONSUMER_PROJECT_ID) $(BUCKET_NAME) $(TOPIC_NAME)

