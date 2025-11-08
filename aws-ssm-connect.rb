class AwsSsmConnect < Formula
  desc "Interactive CLI tool for SSM access via AWS SSO"
  homepage "https://github.com/nkarisa/homebrew-aws-ssm-connect"
  version "v1.0.0"

  # Specify the URL for the source archive (usually a tarball of the release)
  url "https://github.com/nkarisa/homebrew-aws-ssm-connect/archive/v1.0.0.tar.gz"
  # Replace with the actual SHA-256 hash of your v1.0.0 tarball
  sha256 "7323077a159cdffd59e6fa1abf56da7ce06407b270105fab412ba0e8c388450d"

  # Go is used to build the source code
  depends_on "go" => :build

  def install
    # Build the binary using the version tag
    system "go", "build", "-ldflags", "-s -w", "-o", bin/"aws-ssm-connect", "."
  end

  # Test that the binary runs and displays help text
  test do
    assert_match "--- AWS EC2 Instance Lister", shell_output("#{bin}/aws-ssm-connect", 1)
  end
end