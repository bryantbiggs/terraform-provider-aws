package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/hashicorp/terraform-provider-aws/internal/provider/fwprovider"
	"github.com/hashicorp/terraform-provider-aws/version"
)

// ProtoV5ProviderServerFactory returns a muxed terraform-plugin-go protocol v5 provider factory function.
// This factory function is suitable for use with the terraform-plugin-go Serve function.
func ProtoV5ProviderServerFactory(ctx context.Context) (func() tfprotov5.ProviderServer, error) {
	servers := []func() tfprotov5.ProviderServer{
		Provider().GRPCProvider,
		providerserver.NewProtocol5(fwprovider.New(version.ProviderVersion)),
	}

	muxServer, err := tf5muxserver.NewMuxServer(ctx, servers...)

	if err != nil {
		return nil, err
	}

	return muxServer.ProviderServer, nil
}
