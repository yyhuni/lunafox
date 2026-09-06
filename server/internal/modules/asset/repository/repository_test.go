package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRepositoryConstructorsAndHelpers(t *testing.T) {
	db := newAssetRepositoryDB(t)

	if NewDirectoryRepository(db).db != db {
		t.Fatal("expected directory repository to keep db")
	}
	if NewSubdomainRepository(db).db != db {
		t.Fatal("expected subdomain repository to keep db")
	}
	if NewWebsiteRepository(db).db != db {
		t.Fatal("expected website repository to keep db")
	}
	if NewEndpointRepository(db).db != db {
		t.Fatal("expected endpoint repository to keep db")
	}
	if NewHostPortRepository(db).db != db {
		t.Fatal("expected host-port repository to keep db")
	}
	if NewScreenshotRepository(db).db != db {
		t.Fatal("expected screenshot repository to keep db")
	}

	if ExtractHostFromURL("https://example.com:8443/path") != "example.com" {
		t.Fatal("expected hostname to be derived without rewriting URL")
	}
	if ExtractHostFromURL("://bad") != "" {
		t.Fatal("expected invalid URL to return empty host")
	}
}

func TestSubdomainFilterMappingUsesDNSNameBoundaryField(t *testing.T) {
	field, ok := SubdomainFilterMapping["dnsName"]
	if !ok {
		t.Fatal("expected dnsName filter field for subdomain DNS name")
	}
	if field.Column != "dns_name" {
		t.Fatalf("expected dnsName to map to dns_name column, got %q", field.Column)
	}
	if _, ok := SubdomainFilterMapping["name"]; ok {
		t.Fatal("name filter field must not remain a subdomain DNS-name alias")
	}
}

func TestRepositoryMappers(t *testing.T) {
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	status := 200
	contentLength := 512
	directoryContentLength := int64(512)
	directoryDuration := int64(42)
	vhost := true
	status16 := int16(201)

	if directoryModelToDomain(nil) != nil || directoryDomainToModel(nil) != nil {
		t.Fatal("expected nil directory mapper inputs to return nil")
	}
	directory := &assetdomain.Directory{ID: 1, TargetID: 2, URL: "https://example.com/dir", Status: &status, ContentLength: &directoryContentLength, ContentType: "text/html", Duration: &directoryDuration, CreatedAt: now}
	directoryModel := directoryDomainToModel(directory)
	if directoryModel.URL != directory.URL {
		t.Fatalf("unexpected directory model: %+v", directoryModel)
	}
	if got := directoryModelListToDomain(directoryDomainListToModel([]assetdomain.Directory{*directory})); len(got) != 1 || got[0].URL != directory.URL {
		t.Fatalf("unexpected directory list mapping: %+v", got)
	}

	if subdomainModelToDomain(nil) != nil || subdomainDomainToModel(nil) != nil {
		t.Fatal("expected nil subdomain mapper inputs to return nil")
	}
	subdomain := &assetdomain.Subdomain{ID: 3, TargetID: 2, DNSName: "api.example.com", CreatedAt: now}
	if got := subdomainModelListToDomain(subdomainDomainListToModel([]assetdomain.Subdomain{*subdomain})); len(got) != 1 || got[0].DNSName != subdomain.DNSName {
		t.Fatalf("unexpected subdomain list mapping: %+v", got)
	}

	if websiteModelToDomain(nil) != nil || websiteDomainToModel(nil) != nil {
		t.Fatal("expected nil website mapper inputs to return nil")
	}
	website := &assetdomain.Website{
		ID:              4,
		TargetID:        2,
		URL:             "https://example.com",
		Host:            "example.com",
		Location:        "https://example.com/login",
		CreatedAt:       now,
		Title:           "home",
		Webserver:       "nginx",
		ResponseBody:    "ok",
		ContentType:     "text/html",
		Tech:            []string{"gin", "go"},
		StatusCode:      &status,
		ContentLength:   &contentLength,
		Vhost:           &vhost,
		ResponseHeaders: "server: nginx",
	}
	if got := websiteModelListToDomain(websiteDomainListToModel([]assetdomain.Website{*website})); len(got) != 1 || got[0].Host != website.Host {
		t.Fatalf("unexpected website list mapping: %+v", got)
	}

	if endpointModelToDomain(nil) != nil || endpointDomainToModel(nil) != nil {
		t.Fatal("expected nil endpoint mapper inputs to return nil")
	}
	endpoint := &assetdomain.Endpoint{
		ID:                       5,
		TargetID:                 2,
		URL:                      "https://example.com/api",
		Host:                     "example.com",
		Location:                 "https://example.com/login",
		CreatedAt:                now,
		Title:                    "api",
		Webserver:                "nginx",
		ResponseBody:             "ok",
		ResponseBodyTruncated:    true,
		ContentType:              "application/json",
		Tech:                     []string{"gin"},
		StatusCode:               &status,
		ContentLength:            &contentLength,
		Vhost:                    &vhost,
		ResponseHeaders:          "server: nginx",
		ResponseHeadersTruncated: true,
	}
	if got := endpointModelListToDomain(endpointDomainListToModel([]assetdomain.Endpoint{*endpoint})); len(got) != 1 || got[0].URL != endpoint.URL || !got[0].ResponseBodyTruncated || !got[0].ResponseHeadersTruncated {
		t.Fatalf("unexpected endpoint list mapping: %+v", got)
	}

	if hostPortModelToDomain(nil) != nil || hostPortDomainToModel(nil) != nil {
		t.Fatal("expected nil host-port mapper inputs to return nil")
	}
	hostPort := &assetdomain.HostPort{ID: 6, TargetID: 2, Host: "api.example.com", IP: "1.1.1.1", Port: 443, CreatedAt: now}
	if got := hostPortDomainListToModel([]assetdomain.HostPort{*hostPort}); len(got) != 1 || got[0].IP != hostPort.IP {
		t.Fatalf("unexpected host-port list mapping: %+v", got)
	}

	if screenshotModelToDomain(nil) != nil || screenshotDomainToModel(nil) != nil {
		t.Fatal("expected nil screenshot mapper inputs to return nil")
	}
	screenshot := &assetdomain.Screenshot{ID: 7, TargetID: 2, URL: "https://example.com", StatusCode: &status16, Image: []byte("img"), CreatedAt: now, UpdatedAt: now}
	if got := screenshotModelListToDomain(screenshotDomainListToModel([]assetdomain.Screenshot{*screenshot})); len(got) != 1 || string(got[0].Image) != "img" {
		t.Fatalf("unexpected screenshot list mapping: %+v", got)
	}
}

func TestRepositoryQueriesWithSQLite(t *testing.T) {
	db := newAssetRepositoryDB(t)
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	status := 200
	status16 := int16(201)
	length := 256
	vhost := true

	mustCreateAssetFixtures(t, db, now, status, status16, length, vhost)

	directoryRepo := NewDirectoryRepository(db)
	directories, total, err := directoryRepo.ListByTargetID(1, 1, 10, "", "createdAt desc")
	if err != nil {
		t.Fatalf("find directories: %v", err)
	}
	if total != 2 || len(directories) != 2 || directories[0].URL != "https://example.com/b" {
		t.Fatalf("unexpected directories: total=%d items=%+v", total, directories)
	}
	var streamedDirectories []assetdomain.Directory
	if err := directoryRepo.ForEachByTargetID(context.Background(), 1, func(item assetdomain.Directory) error {
		streamedDirectories = append(streamedDirectories, item)
		return nil
	}); err != nil || len(streamedDirectories) == 0 || streamedDirectories[0].URL != "https://example.com/b" {
		t.Fatalf("unexpected streamed directories items=%+v err=%v", streamedDirectories, err)
	}
	directoryCount, err := directoryRepo.CountByTargetID(1)
	if err != nil || directoryCount != 2 {
		t.Fatalf("unexpected directory count=%d err=%v", directoryCount, err)
	}

	subdomainRepo := NewSubdomainRepository(db)
	subdomains, total, err := subdomainRepo.ListByTargetID(1, 1, 10, "", "")
	if err != nil {
		t.Fatalf("find subdomains: %v", err)
	}
	if total != 2 || len(subdomains) != 2 || subdomains[0].DNSName != "api.example.com" {
		t.Fatalf("unexpected subdomains: total=%d items=%+v", total, subdomains)
	}
	var streamedSubdomains []assetdomain.Subdomain
	if err := subdomainRepo.ForEachByTargetID(context.Background(), 1, func(item assetdomain.Subdomain) error {
		streamedSubdomains = append(streamedSubdomains, item)
		return nil
	}); err != nil || len(streamedSubdomains) == 0 || streamedSubdomains[0].DNSName != "api.example.com" {
		t.Fatalf("unexpected streamed subdomains items=%+v err=%v", streamedSubdomains, err)
	}
	subdomainCount, err := subdomainRepo.CountByTargetID(1)
	if err != nil || subdomainCount != 2 {
		t.Fatalf("unexpected subdomain count=%d err=%v", subdomainCount, err)
	}

	websiteRepo := NewWebsiteRepository(db)
	websites, total, err := websiteRepo.ListByTargetID(1, 1, 10, "", "createdAt desc")
	if err != nil {
		t.Fatalf("find websites: %v", err)
	}
	if total != 2 || len(websites) != 2 || websites[0].URL != "https://example.com/b" {
		t.Fatalf("unexpected websites: total=%d items=%+v", total, websites)
	}
	websiteItem, err := websiteRepo.GetByID(31)
	if err != nil || websiteItem.URL != "https://example.com/a" {
		t.Fatalf("unexpected website item=%+v err=%v", websiteItem, err)
	}
	var streamedWebsites []assetdomain.Website
	if err := websiteRepo.ForEachByTargetID(context.Background(), 1, func(item assetdomain.Website) error {
		streamedWebsites = append(streamedWebsites, item)
		return nil
	}); err != nil || len(streamedWebsites) == 0 || streamedWebsites[0].URL != "https://example.com/b" {
		t.Fatalf("unexpected streamed websites items=%+v err=%v", streamedWebsites, err)
	}
	websiteCount, err := websiteRepo.CountByTargetID(1)
	if err != nil || websiteCount != 2 {
		t.Fatalf("unexpected website count=%d err=%v", websiteCount, err)
	}

	endpointRepo := NewEndpointRepository(db)
	endpoints, total, err := endpointRepo.ListByTargetID(1, 1, 10, "", "createdAt desc")
	if err != nil {
		t.Fatalf("find endpoints: %v", err)
	}
	if total != 2 || len(endpoints) != 2 || endpoints[0].URL != "https://example.com/api/v2" {
		t.Fatalf("unexpected endpoints: total=%d items=%+v", total, endpoints)
	}
	endpointItem, err := endpointRepo.GetByID(41)
	if err != nil || endpointItem.URL != "https://example.com/api/v1" {
		t.Fatalf("unexpected endpoint item=%+v err=%v", endpointItem, err)
	}
	var streamedEndpoints []assetdomain.Endpoint
	if err := endpointRepo.ForEachByTargetID(context.Background(), 1, func(item assetdomain.Endpoint) error {
		streamedEndpoints = append(streamedEndpoints, item)
		return nil
	}); err != nil || len(streamedEndpoints) == 0 || streamedEndpoints[0].URL != "https://example.com/api/v2" {
		t.Fatalf("unexpected streamed endpoints items=%+v err=%v", streamedEndpoints, err)
	}
	endpointCount, err := endpointRepo.CountByTargetID(1)
	if err != nil || endpointCount != 2 {
		t.Fatalf("unexpected endpoint count=%d err=%v", endpointCount, err)
	}

	screenshotRepo := NewScreenshotRepository(db)
	screenshots, total, err := screenshotRepo.ListByTargetID(1, 1, 10, "", "createdAt desc")
	if err != nil {
		t.Fatalf("find screenshots: %v", err)
	}
	if total != 2 || len(screenshots) != 2 || screenshots[0].URL != "https://example.com/b" {
		t.Fatalf("unexpected screenshots: total=%d items=%+v", total, screenshots)
	}
	screenshotItem, err := screenshotRepo.GetByID(51)
	if err != nil || screenshotItem.URL != "https://example.com/a" {
		t.Fatalf("unexpected screenshot item=%+v err=%v", screenshotItem, err)
	}

	hostPortRepo := NewHostPortRepository(db)
	ipRows, total, err := hostPortRepo.GetIPAggregation(1, 1, 10, "", "createdAt desc")
	if err == nil {
		t.Fatalf("expected sqlite aggregate scan error, got rows=%+v total=%d", ipRows, total)
	}
	hosts, ports, err := hostPortRepo.GetHostsAndPortsByIP(1, "1.1.1.1", "")
	if err != nil {
		t.Fatalf("get hosts and ports: %v", err)
	}
	if len(hosts) != 2 || hosts[0] != "a.example.com" || len(ports) != 2 || ports[0] != 80 {
		t.Fatalf("unexpected hosts=%v ports=%v", hosts, ports)
	}
	var streamedHostPorts []assetdomain.HostPort
	if err := hostPortRepo.ForEachByTargetID(context.Background(), 1, func(item assetdomain.HostPort) error {
		streamedHostPorts = append(streamedHostPorts, item)
		return nil
	}); err != nil || len(streamedHostPorts) == 0 || streamedHostPorts[0].IP != "1.1.1.1" || streamedHostPorts[0].Host != "a.example.com" {
		t.Fatalf("unexpected streamed host ports items=%+v err=%v", streamedHostPorts, err)
	}
	var filteredHostPorts []assetdomain.HostPort
	if err := hostPortRepo.ForEachByTargetIDAndIPs(context.Background(), 1, []string{"2.2.2.2"}, func(item assetdomain.HostPort) error {
		filteredHostPorts = append(filteredHostPorts, item)
		return nil
	}); err != nil || len(filteredHostPorts) == 0 || filteredHostPorts[0].IP != "2.2.2.2" {
		t.Fatalf("unexpected filtered host ports items=%+v err=%v", filteredHostPorts, err)
	}
	hostPortCount, err := hostPortRepo.CountByTargetID(1)
	if err != nil || hostPortCount != 2 {
		t.Fatalf("unexpected host-port count=%d err=%v", hostPortCount, err)
	}
}

func TestSubdomainRepositoryListByTargetUsesStableServerOrder(t *testing.T) {
	db := newAssetRepositoryDB(t)
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	status := 200
	status16 := int16(201)
	length := 256
	vhost := true
	mustCreateAssetFixtures(t, db, now, status, status16, length, vhost)

	repo := NewSubdomainRepository(db)

	byNameAsc, _, err := repo.ListByTargetID(1, 1, 10, "", "dnsName")
	if err != nil {
		t.Fatalf("list dnsName asc: %v", err)
	}
	if len(byNameAsc) != 2 || byNameAsc[0].DNSName != "api.example.com" || byNameAsc[1].DNSName != "www.example.com" {
		t.Fatalf("unexpected dnsName asc order: %+v", byNameAsc)
	}

	byCreatedAtAsc, _, err := repo.ListByTargetID(1, 1, 10, "", "createdAt asc")
	if err != nil {
		t.Fatalf("list createdAt asc: %v", err)
	}
	if len(byCreatedAtAsc) != 2 || byCreatedAtAsc[0].DNSName != "www.example.com" || byCreatedAtAsc[1].DNSName != "api.example.com" {
		t.Fatalf("unexpected createdAt asc order: %+v", byCreatedAtAsc)
	}

}

func TestRepositoryCommandsWithSQLite(t *testing.T) {
	status := 200
	length := 128
	directoryLength := int64(128)
	directoryDuration := int64(60)
	status16 := int16(204)
	vhost := true

	t.Run("directory", func(t *testing.T) {
		db := newAssetRepositoryDB(t)
		repo := NewDirectoryRepository(db)

		if created, err := repo.BatchCreate(nil); err != nil || created != 0 {
			t.Fatalf("unexpected empty batch create result created=%d err=%v", created, err)
		}
		created, err := repo.BatchCreate([]assetdomain.Directory{
			{TargetID: 1, URL: "https://example.com/dir-a"},
			{TargetID: 1, URL: "https://example.com/dir-b"},
		})
		if err != nil || created != 2 {
			t.Fatalf("unexpected batch create result created=%d err=%v", created, err)
		}

		var ids []int
		if err := db.Model(&model.Directory{}).Pluck("id", &ids).Error; err != nil {
			t.Fatalf("pluck directory ids: %v", err)
		}
		if deleted, err := repo.BatchDelete(nil); err != nil || deleted != 0 {
			t.Fatalf("unexpected empty batch delete result deleted=%d err=%v", deleted, err)
		}
		deleted, err := repo.BatchDelete([]int{ids[0]})
		if err != nil || deleted != 1 {
			t.Fatalf("unexpected batch delete result deleted=%d err=%v", deleted, err)
		}

		if affected, err := repo.BatchUpsert(nil); err != nil || affected != 0 {
			t.Fatalf("unexpected empty batch upsert result affected=%d err=%v", affected, err)
		}
		if affected, err := repo.upsertBatch(nil); err != nil || affected != 0 {
			t.Fatalf("unexpected empty upsertBatch result affected=%d err=%v", affected, err)
		}

		affected, err := repo.BatchUpsert([]assetdomain.Directory{{
			TargetID:      1,
			URL:           "https://example.com/dir-upsert",
			Status:        &status,
			ContentLength: &directoryLength,
			ContentType:   "text/html",
			Duration:      &directoryDuration,
		}})
		if err != nil || affected == 0 {
			t.Fatalf("unexpected batch upsert result affected=%d err=%v", affected, err)
		}
	})

	t.Run("subdomain", func(t *testing.T) {
		db := newAssetRepositoryDB(t)
		repo := NewSubdomainRepository(db)

		if created, err := repo.BatchCreate(nil); err != nil || created != 0 {
			t.Fatalf("unexpected empty batch create result created=%d err=%v", created, err)
		}
		created, err := repo.BatchCreate([]assetdomain.Subdomain{
			{TargetID: 1, DNSName: "api.example.com"},
			{TargetID: 1, DNSName: "www.example.com"},
		})
		if err != nil || created != 2 {
			t.Fatalf("unexpected batch create result created=%d err=%v", created, err)
		}

		var ids []int
		if err := db.Model(&model.Subdomain{}).Pluck("id", &ids).Error; err != nil {
			t.Fatalf("pluck subdomain ids: %v", err)
		}
		if deleted, err := repo.BatchDelete(nil); err != nil || deleted != 0 {
			t.Fatalf("unexpected empty batch delete result deleted=%d err=%v", deleted, err)
		}
		deleted, err := repo.BatchDelete([]int{ids[0]})
		if err != nil || deleted != 1 {
			t.Fatalf("unexpected batch delete result deleted=%d err=%v", deleted, err)
		}
	})

	t.Run("website", func(t *testing.T) {
		db := newAssetRepositoryDB(t)
		repo := NewWebsiteRepository(db)

		if created, err := repo.BatchCreate(nil); err != nil || created != 0 {
			t.Fatalf("unexpected empty batch create result created=%d err=%v", created, err)
		}
		created, err := repo.BatchCreate([]assetdomain.Website{
			{TargetID: 1, URL: "https://example.com/a", Host: "example.com"},
			{TargetID: 1, URL: "https://example.com/b", Host: "example.com"},
		})
		if err != nil || created != 2 {
			t.Fatalf("unexpected batch create result created=%d err=%v", created, err)
		}

		var ids []int
		if err := db.Model(&model.Website{}).Pluck("id", &ids).Error; err != nil {
			t.Fatalf("pluck website ids: %v", err)
		}
		if err := repo.Delete(ids[0]); err != nil {
			t.Fatalf("delete website: %v", err)
		}
		if deleted, err := repo.BatchDelete(nil); err != nil || deleted != 0 {
			t.Fatalf("unexpected empty batch delete result deleted=%d err=%v", deleted, err)
		}
		deleted, err := repo.BatchDelete([]int{ids[1]})
		if err != nil || deleted != 1 {
			t.Fatalf("unexpected batch delete result deleted=%d err=%v", deleted, err)
		}

		if affected, err := repo.BatchUpsert(nil); err != nil || affected != 0 {
			t.Fatalf("unexpected empty batch upsert result affected=%d err=%v", affected, err)
		}
		if affected, err := repo.upsertBatch(nil); err != nil || affected != 0 {
			t.Fatalf("unexpected empty upsertBatch result affected=%d err=%v", affected, err)
		}
		affected, err := repo.BatchUpsert([]assetdomain.Website{{
			TargetID:        1,
			URL:             "https://example.com/upsert",
			Host:            "example.com",
			Title:           "title",
			Webserver:       "nginx",
			ContentType:     "text/html",
			StatusCode:      &status,
			ContentLength:   &length,
			Vhost:           &vhost,
			ResponseHeaders: "server: nginx",
		}})
		if err == nil && affected == 0 {
			t.Fatalf("expected batch upsert to affect rows when no error, got affected=%d", affected)
		}
	})

	t.Run("endpoint", func(t *testing.T) {
		db := newAssetRepositoryDB(t)
		repo := NewEndpointRepository(db)

		if created, err := repo.BatchCreate(nil); err != nil || created != 0 {
			t.Fatalf("unexpected empty batch create result created=%d err=%v", created, err)
		}
		created, err := repo.BatchCreate([]assetdomain.Endpoint{
			{TargetID: 1, URL: "https://example.com/api/a", Host: "example.com"},
			{TargetID: 1, URL: "https://example.com/api/b", Host: "example.com"},
		})
		if err != nil || created != 2 {
			t.Fatalf("unexpected batch create result created=%d err=%v", created, err)
		}

		var ids []int
		if err := db.Model(&model.Endpoint{}).Pluck("id", &ids).Error; err != nil {
			t.Fatalf("pluck endpoint ids: %v", err)
		}
		if err := repo.Delete(ids[0]); err != nil {
			t.Fatalf("delete endpoint: %v", err)
		}
		if deleted, err := repo.BatchDelete(nil); err != nil || deleted != 0 {
			t.Fatalf("unexpected empty batch delete result deleted=%d err=%v", deleted, err)
		}
		deleted, err := repo.BatchDelete([]int{ids[1]})
		if err != nil || deleted != 1 {
			t.Fatalf("unexpected batch delete result deleted=%d err=%v", deleted, err)
		}

		if affected, err := repo.BatchUpsert(nil); err != nil || affected != 0 {
			t.Fatalf("unexpected empty batch upsert result affected=%d err=%v", affected, err)
		}
		if affected, err := repo.upsertBatch(nil); err != nil || affected != 0 {
			t.Fatalf("unexpected empty upsertBatch result affected=%d err=%v", affected, err)
		}
		affected, err := repo.BatchUpsert([]assetdomain.Endpoint{{
			TargetID:        1,
			URL:             "https://example.com/api/upsert",
			Host:            "example.com",
			Title:           "title",
			Webserver:       "nginx",
			ContentType:     "application/json",
			StatusCode:      &status,
			ContentLength:   &length,
			Vhost:           &vhost,
			ResponseHeaders: "server: nginx",
		}})
		if err == nil && affected == 0 {
			t.Fatalf("expected batch upsert to affect rows when no error, got affected=%d", affected)
		}
	})

	t.Run("screenshot", func(t *testing.T) {
		db := newAssetRepositoryDB(t)
		repo := NewScreenshotRepository(db)

		if err := db.Create(&model.Screenshot{ID: 61, TargetID: 1, URL: "https://example.com/a"}).Error; err != nil {
			t.Fatalf("seed screenshot: %v", err)
		}
		if deleted, err := repo.BatchDelete(nil); err != nil || deleted != 0 {
			t.Fatalf("unexpected empty batch delete result deleted=%d err=%v", deleted, err)
		}
		deleted, err := repo.BatchDelete([]int{61})
		if err != nil || deleted != 1 {
			t.Fatalf("unexpected batch delete result deleted=%d err=%v", deleted, err)
		}
		if affected, err := repo.BatchUpsert(nil); err != nil || affected != 0 {
			t.Fatalf("unexpected empty batch upsert result affected=%d err=%v", affected, err)
		}
		affected, err := repo.BatchUpsert([]assetdomain.Screenshot{{
			TargetID:   1,
			URL:        "https://example.com/b",
			StatusCode: &status16,
			Image:      []byte("img"),
		}})
		if err != nil || affected == 0 {
			t.Fatalf("unexpected screenshot batch upsert result affected=%d err=%v", affected, err)
		}
	})

	t.Run("host port", func(t *testing.T) {
		db := newAssetRepositoryDB(t)
		repo := NewHostPortRepository(db)

		if affected, err := repo.BatchUpsert(nil); err != nil || affected != 0 {
			t.Fatalf("unexpected empty batch upsert result affected=%d err=%v", affected, err)
		}
		affected, err := repo.BatchUpsert([]assetdomain.HostPort{
			{TargetID: 1, Host: "a.example.com", IP: "1.1.1.1", Port: 80},
			{TargetID: 1, Host: "b.example.com", IP: "2.2.2.2", Port: 443},
		})
		if err != nil || affected != 2 {
			t.Fatalf("unexpected host-port batch upsert result affected=%d err=%v", affected, err)
		}
		if deleted, err := repo.DeleteByIPs(nil); err != nil || deleted != 0 {
			t.Fatalf("unexpected empty delete by ips result deleted=%d err=%v", deleted, err)
		}
		deleted, err := repo.DeleteByIPs([]string{"1.1.1.1"})
		if err != nil || deleted != 1 {
			t.Fatalf("unexpected delete by ips result deleted=%d err=%v", deleted, err)
		}
	})
}

func newAssetRepositoryDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	schema := []string{
		`CREATE TABLE target (
			id INTEGER PRIMARY KEY,
			name TEXT,
			type TEXT,
			created_at DATETIME,
			last_scanned_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE directory (
			id INTEGER PRIMARY KEY,
			target_id INTEGER NOT NULL,
			url TEXT NOT NULL,
			status INTEGER,
			content_length INTEGER,
			content_type TEXT,
			duration INTEGER,
			created_at DATETIME,
			UNIQUE(target_id, url)
		)`,
		`CREATE TABLE subdomain (
			id INTEGER PRIMARY KEY,
			target_id INTEGER NOT NULL,
			dns_name TEXT NOT NULL,
			created_at DATETIME,
			UNIQUE(target_id, dns_name)
		)`,
		`CREATE TABLE website (
			id INTEGER PRIMARY KEY,
			target_id INTEGER NOT NULL,
			url TEXT NOT NULL,
			host TEXT,
			location TEXT,
			created_at DATETIME,
			title TEXT,
			webserver TEXT,
			response_body TEXT,
			content_type TEXT,
			tech TEXT,
			status_code INTEGER,
			content_length INTEGER,
			vhost NUMERIC,
			response_headers TEXT,
			UNIQUE(target_id, url)
		)`,
		`CREATE TABLE endpoint (
			id INTEGER PRIMARY KEY,
			target_id INTEGER NOT NULL,
			url TEXT NOT NULL,
			host TEXT,
			location TEXT,
			created_at DATETIME,
			title TEXT,
			webserver TEXT,
			response_body TEXT,
			response_body_truncated NUMERIC NOT NULL DEFAULT 0,
			content_type TEXT,
			tech TEXT,
			status_code INTEGER,
			content_length INTEGER,
			vhost NUMERIC,
			response_headers TEXT,
			response_headers_truncated NUMERIC NOT NULL DEFAULT 0,
			UNIQUE(target_id, url)
		)`,
		`CREATE TABLE screenshot (
			id INTEGER PRIMARY KEY,
			target_id INTEGER NOT NULL,
			url TEXT NOT NULL,
			status_code INTEGER,
			image BLOB,
			created_at DATETIME,
			updated_at DATETIME,
			UNIQUE(target_id, url)
		)`,
		`CREATE TABLE host_port_mapping (
			id INTEGER PRIMARY KEY,
			target_id INTEGER NOT NULL,
			host TEXT NOT NULL,
			ip TEXT NOT NULL,
			port INTEGER NOT NULL,
			created_at DATETIME,
			UNIQUE(target_id, host, ip, port)
		)`,
	}
	for _, stmt := range schema {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create schema failed: %v", err)
		}
	}
	if err := db.Create(&model.AssetTargetRef{ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}).Error; err != nil {
		t.Fatalf("seed target: %v", err)
	}
	return db
}

func mustCreateAssetFixtures(t *testing.T, db *gorm.DB, now time.Time, status int, status16 int16, length int, vhost bool) {
	t.Helper()

	if err := db.Create([]model.Directory{
		{ID: 11, TargetID: 1, URL: "https://example.com/a", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: 12, TargetID: 1, URL: "https://example.com/b", CreatedAt: now.Add(-1 * time.Hour)},
	}).Error; err != nil {
		t.Fatalf("seed directories: %v", err)
	}
	if err := db.Create([]model.Subdomain{
		{ID: 21, TargetID: 1, DNSName: "www.example.com", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: 22, TargetID: 1, DNSName: "api.example.com", CreatedAt: now.Add(-1 * time.Hour)},
	}).Error; err != nil {
		t.Fatalf("seed subdomains: %v", err)
	}
	if err := db.Create([]model.Website{
		{ID: 31, TargetID: 1, URL: "https://example.com/a", Host: "example.com", StatusCode: &status, ContentLength: &length, Vhost: &vhost, CreatedAt: now.Add(-2 * time.Hour), Tech: pq.StringArray{}},
		{ID: 32, TargetID: 1, URL: "https://example.com/b", Host: "example.com", StatusCode: &status, ContentLength: &length, Vhost: &vhost, CreatedAt: now.Add(-1 * time.Hour), Tech: pq.StringArray{}},
	}).Error; err != nil {
		t.Fatalf("seed websites: %v", err)
	}
	if err := db.Create([]model.Endpoint{
		{ID: 41, TargetID: 1, URL: "https://example.com/api/v1", Host: "example.com", StatusCode: &status, ContentLength: &length, Vhost: &vhost, CreatedAt: now.Add(-2 * time.Hour), Tech: pq.StringArray{}},
		{ID: 42, TargetID: 1, URL: "https://example.com/api/v2", Host: "example.com", StatusCode: &status, ContentLength: &length, Vhost: &vhost, CreatedAt: now.Add(-1 * time.Hour), Tech: pq.StringArray{}},
	}).Error; err != nil {
		t.Fatalf("seed endpoints: %v", err)
	}
	if err := db.Create([]model.Screenshot{
		{ID: 51, TargetID: 1, URL: "https://example.com/a", StatusCode: &status16, Image: []byte("a"), CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: 52, TargetID: 1, URL: "https://example.com/b", StatusCode: &status16, Image: []byte("b"), CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
	}).Error; err != nil {
		t.Fatalf("seed screenshots: %v", err)
	}
	if err := db.Create([]model.HostPort{
		{ID: 61, TargetID: 1, Host: "a.example.com", IP: "1.1.1.1", Port: 80, CreatedAt: now.Add(-3 * time.Hour)},
		{ID: 62, TargetID: 1, Host: "b.example.com", IP: "1.1.1.1", Port: 443, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: 63, TargetID: 1, Host: "c.example.com", IP: "2.2.2.2", Port: 8080, CreatedAt: now.Add(-1 * time.Hour)},
	}).Error; err != nil {
		t.Fatalf("seed hostPorts: %v", err)
	}
}
