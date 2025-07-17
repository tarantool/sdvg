# Goals and Standards Compliance

## Table of Contents

- [Tool Benefits](#tool-benefits)
- [User Stories](#user-stories)
- [VK Tech Values](#vk-tech-values)
- [CNCF Standards Compliance](#cncf-standards-compliance)

## Tool Benefits

With the rise of tools for working with large datasets (millions of rows) and large language models (LLMs),
the need for generating datasets has become more relevant than ever. However, most existing tools focus on generating
text for training LLMs and don’t address the need to create datasets for database benchmarks.
Those that do often suffer from poor usability, limited features, or low performance. That’s why we created our tool -
to cover a wide range of use cases and applications:

### Testing and Debugging

**Algorithm testing:**
Developers can use synthetic data to test and debug their algorithms without having to wait for real data.

**Automated testing:**
The tool can be integrated into CI/CD pipelines for automated testing of applications and services.

### Data Analysis and Visualization

**Exploratory Data Analysis (EDA):**
Synthetic data can be used for preliminary analysis and hypothesis testing before working with real data.

**Data visualization:**
Generating data with specific characteristics makes it easy to create illustrative examples for educational materials
and demos.

### Data Protection and Privacy

**Sensitive data protection:**
Synthetic data can replace real data for testing and analysis, helping protect confidential information.

**Attack simulation:**
You can generate data with certain anomalies or errors to simulate attacks and test system resilience.

### Business and Industry

**Product development:**
Companies can use synthetic data to test and develop new products and services.

**Market research:**
Generating consumer and market data can help with market research and strategic planning.

### Flexibility and Customization

**Custom conditions:**
You can set specific generation conditions to create datasets that match the requirements of a particular task.

**Modularity and extensibility:**
Open source allows other developers to contribute improvements and add new features, making the tool more versatile and
powerful.

## User Stories

### User Story 1

**Actor:** Developer

**Situation:**
A developer is working on a project that needs test data to validate application functionality.
They need to quickly generate a dataset and save it to a file for later use.

**Action:**
The developer uses the CLI interface to set generation parameters (e.g., number of records, data format) and saves the
result to a local file.

**Result:**
The file with the generated data is successfully created in the specified directory, and the data matches the defined
parameters.

**Acceptance Criteria:**

- The file is created in the specified directory.
- Data format and structure match the defined parameters.
- Clear error messages are shown if there are invalid parameters.
- The tool handles different data formats correctly (JSON, CSV, etc.).

### User Story 2

**Actor:** System

**Situation:**
A data generator needs to be integrated with another HTTP service (e.g., a test cluster) to automatically upload
generated data for testing.

**Action:**
An integration service calls the data generator via the API, gets the generated data, and sends it to another HTTP
service for ingestion.

**Result:**
Generated data is successfully uploaded to the target HTTP service, providing up-to-date test data for further use.

**Acceptance Criteria:**

- Data is correctly generated and transferred to the target HTTP service.
- Integration works without failures or errors.
- Errors during data transfer are handled gracefully, with retries.
- Logs are available for successful and failed operations.

### User Story 3

**Actor:** Test Engineer

**Situation:**
A test engineer needs to run load tests for a new app release.
This requires generating large volumes of test data automatically as part of the CI/CD process.

**Action:**
The engineer configures the CI/CD pipeline so that during the preparation stage for load testing, the data generator is
automatically triggered.
The generator creates the required amount of data, which is then used in load tests.

**Result:**
The CI/CD pipeline successfully generates and uses large volumes of test data to run load tests, verifying application
performance under high load.

**Acceptance Criteria:**

- The data generator is integrated into the CI/CD pipeline and runs automatically during each test.
- The generated data matches the required parameters and volume for load testing.
- The data generation process doesn’t slow down the pipeline and stays within acceptable time limits.
- In case of errors during data generation or integration, the pipeline provides informative logs for debugging.

## VK Tech Values

In this project, we focused on incorporating the values of our engineering culture:

**1. Collaboration and open communication**

The project is part of InnerSource and open to contributors - we maintain a list of possible improvements for everyone
interested.

**2. Continuous learning**

Working on this project, we aimed to avoid repeating what we’d done before, using new approaches and resources instead.

**3. Striving for excellence**

From the start, we built a clean architecture foundation and chose a robust set of linters.

**4. Quality first**

We deliberately limited the scope to focus on delivering higher quality in a smaller feature set.

**5. Innovation**

The tool is cloud-native and compatible with LLMs.

**6. Ownership and responsibility**

We plan to keep developing the tool beyond the hackathon because it’s genuinely useful in our daily work.

**7. Customer focus**

Our main audience is developers.
Developers in presale and professional services can deliver better testing results using this tool.
Product teams can test new features in conditions closer to real scenarios.

## CNCF Standards Compliance

Our data generator complies with Cloud Native Computing Foundation (CNCF) standards,
ensuring reliability, scalability, and ease of use in modern cloud environments.

### Containerization

The generator is containerized with Docker for portability and isolation.
This makes it easy to deploy across different environments without dependency issues.

### Microservice Architecture

The generator itself is a microservice that can be used as part of a larger service.

### API-first Approach

Our HTTP API follows REST standards, making integration with other systems straightforward and easy to use.

### Continuous Integration and Delivery (CI/CD)

We’ve implemented CI/CD processes that automate testing, allowing us to deploy changes quickly and safely.

### Scalability and Fault Tolerance

Our service supports horizontal scaling to handle changing loads,
with mechanisms for automatic recovery in case of failures.
