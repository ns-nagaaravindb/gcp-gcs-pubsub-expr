.PHONY: help terraform-init terraform-plan terraform-apply terraform-destroy terraform-output \
	app-build app-create-file app-listen app-test clean \
	check-deps check-gcloud check-terraform check-go

# Configuration
PROJECT_ID ?= $(shell gcloud config get-value project 2>/dev/null)
BUCKET_NAME ?= test-dp-gcspubsub-bucket
TOPIC_NAME ?= test-dp-gcspubsub-bucket
REGION ?= us-central1
TERRAFORM_DIR = pubsub-notification-terraform
APP_DIR = app
APP_BINARY = gcs-pubsub-app

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
	cd $(TERRAFORM_DIR) && terraform plan \
		-var="project_id=$(PROJECT_ID)" \
		-var="region=$(REGION)" \
		-var="bucket_name=$(BUCKET_NAME)" \
		-var="topic_name=$(TOPIC_NAME)"

terraform-apply: terraform-init ## Apply Terraform configuration
	@if [ -z "$(PROJECT_ID)" ]; then \
		echo "$(RED)Error: PROJECT_ID not set. Set it via: make terraform-apply PROJECT_ID=your-project-id$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Applying Terraform configuration...$(NC)"
	cd $(TERRAFORM_DIR) && terraform apply \
		-var="project_id=$(PROJECT_ID)" \
		-var="region=$(REGION)" \
		-var="bucket_name=$(BUCKET_NAME)" \
		-var="topic_name=$(TOPIC_NAME)" \
		-auto-approve
	@echo "$(GREEN)✓ Infrastructure created successfully$(NC)"

terraform-destroy: terraform-init ## Destroy Terraform infrastructure
	@if [ -z "$(PROJECT_ID)" ]; then \
		echo "$(RED)Error: PROJECT_ID not set. Set it via: make terraform-destroy PROJECT_ID=your-project-id$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Destroying Terraform infrastructure...$(NC)"
	cd $(TERRAFORM_DIR) && terraform destroy \
		-var="project_id=$(PROJECT_ID)" \
		-var="region=$(REGION)" \
		-var="bucket_name=$(BUCKET_NAME)" \
		-var="topic_name=$(TOPIC_NAME)" \
		-auto-approve
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
	./$(APP_BINARY) \
		-mode=listen \
		-project=$(PROJECT_ID) \
		-bucket=$(BUCKET_NAME) \
		-topic=$(TOPIC_NAME)

app-test: app-build ## Create file and listen for notification
	@if [ -z "$(PROJECT_ID)" ]; then \
		echo "$(RED)Error: PROJECT_ID not set. Set it via: make app-test PROJECT_ID=your-project-id$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Creating file and listening for notification...$(NC)"
	./$(APP_BINARY) \
		-mode=both \
		-project=$(PROJECT_ID) \
		-bucket=$(BUCKET_NAME) \
		-topic=$(TOPIC_NAME) \
		-file=test-$(shell date +%s).txt \
		-content="Test file created at $(shell date)"

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
	@echo "  BUCKET_NAME: $(BUCKET_NAME)"
	@echo "  TOPIC_NAME: $(TOPIC_NAME)"
	@echo "  REGION: $(REGION)"

test-infra: terraform-plan ## Test infrastructure configuration without applying
	@echo "$(GREEN)✓ Infrastructure configuration is valid$(NC)"

