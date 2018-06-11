package main

/*
func validateTunnelOptions(t *testing.T, po *programOptions, to *tunnelOptions) {
	assert.Equal(t, po.WireGuard.ServerPrivateKey, to.WireguardPrivateKey, "Wireguard: private key mismatch")
	assert.Equal(t, po.WireGuard.Enable, to.WireguardEnable, "Wireguard: enable mismatch")
}

func TestTunnelOptionsGeneration(t *testing.T) {
	p := &programOptions{}
	p.WireGuard.Enable = true
	p.WireGuard.ServerPrivateKey = "MBaaA+HgkX4EGbtFmN5A2GY9aEZxc6cdaMZnIZ9o02o="
	p.WireGuard.PeerPublicKeys = []string{
		"9HYUqL2BAGjzTDLdSeIEMxX4Jk4dNVOon8ugjVuGkHU=",
		"vWErPBtP/rVm9U/6TOwodZa7f+7HEJoqahNtjYLA2BA=",
		"pjBONJhr8Vzw0RYJOdfxFq4VqCyuaX3EY7P9ZOrHmiI=",
		"uhZiMI66LpeZsW+jXfLiTKpYY2N9rAoUb3UAlI9yWRo=",
		"1hvlzjSb+Ju3JBoOyNsleoDjVMFgvB9/FrINELvCBgo=",
	}
	p.WireGuard.UseStaticPort = true
	p.WireGuard.StaticPort = 55000
	p.ObfsproxyIPv4.Enable = true
	p.ObfsproxyIPv4.Secret = "MI3WYVCBMVLGS4TFIZYDMOKBNVLUM43Y"
	p.ObfsproxyIPv4.UseStaticPort = true
	p.ObfsproxyIPv4.StaticPort = 56000
	p.ObfsproxyIPv6.Enable = true
	p.ObfsproxyIPv6.Secret = "INVWYTBYKJXDA6CFLFSVGVDSPJFWSUKJ"
	p.ObfsproxyIPv6.UseStaticPort = true
	p.ObfsproxyIPv6.StaticPort = 57000

	opts, err := newTunnelOptions(p)
	assert.NoError(t, err)
	assert.NotNil(t, opts)
}
*/
