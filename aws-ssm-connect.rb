class AwsSsmConnect < Formula
  desc "Interactive CLI tool for SSM access via AWS SSO"
  homepage "https://github.com/nkarisa/homebrew-aws-ssm-connect"
  version "v1.0.0"
  
  # Specify the URL for the source archive (usually a tarball of the release)
  url "https://github.com/nkarisa/aws-ssm-connect/v1.0.0.tar.gz"
  # Replace with the actual SHA-256 hash of your v1.0.0 tarball
  sha256 "7c151c74a016b0e41308f61425571563f7447778"

  # Go is used to build the source code
  depends_on "go" => :build

  def install
    # Build the binary using the version tag
    system "go", "build", "-ldflags", "-s -w", "-o", bin/"ec2-lister", "."
  end

  # Test that the binary runs and displays help text
  test do
    assert_match "--- AWS EC2 Instance Lister", shell_output("#{bin}/aws-ssm-connect", 1)
  end
end