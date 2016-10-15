package csp

/*
#include "common.h"
HCERTSTORE openStoreMem() {
	return CertOpenStore(CERT_STORE_PROV_MEMORY, MY_ENC_TYPE, 0, CERT_STORE_CREATE_NEW_FLAG, NULL);
}


HCERTSTORE openStoreSystem(HCRYPTPROV hProv, CHAR *proto) {
	return CertOpenStore(
		CERT_STORE_PROV_SYSTEM_A,          // The store provider type
		0,                               // The encoding type is
		// not needed
		hProv,                            // Use the default HCRYPTPROV
		// Set the store location in a
		// registry location
		CERT_STORE_NO_CRYPT_RELEASE_FLAG | CERT_SYSTEM_STORE_CURRENT_USER,
		proto                            // The store name as a Unicode
		// string
	);
}

*/
import "C"

import (
	"encoding/hex"
	//"io"
	//"io/ioutil"
	//"fmt"
	"unsafe"
)

// CertStore incapsulates certificate store
type CertStore struct {
	hStore C.HCERTSTORE
}

// MemoryStore returns handle to new empty in-memory certificate store
func MemoryStore() (res CertStore, err error) {
	res.hStore = C.openStoreMem()
	if res.hStore == C.HCERTSTORE(nil) {
		err = getErr("Error creating memory cert store")
		return
	}
	return
}

// SystemStore returns handle to certificate store with certain name, using
// default system cryptoprovider
func SystemStore(name string) (res CertStore, err error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	res.hStore = C.openStoreSystem(C.HCRYPTPROV(0), (*C.CHAR)(cName))
	if res.hStore == C.HCERTSTORE(nil) {
		err = getErr("Error getting system cert store")
		return
	}
	return
}

// CertStore method returns handle to certificate store in certain CSP context
func (c Ctx) CertStore(name string) (res CertStore, err error) {
	cName := charPtr(name)
	defer freePtr(cName)

	res.hStore = C.openStoreSystem(c.hProv, cName)
	if res.hStore == nil {
		err = getErr("Error getting system cert store")
		return
	}
	return
}

// Close releases cert store handle
func (s CertStore) Close() error {
	if C.CertCloseStore(s.hStore, C.CERT_CLOSE_STORE_CHECK_FLAG) == 0 {
		return getErr("Error closing cert store")
	}
	return nil
}

// FindCerts returns slice of *Cert's in store that satisfy findType and findPara
func (s CertStore) FindCerts(findType C.DWORD, findPara unsafe.Pointer) []Cert {
	var res []Cert

	for pCert := C.CertFindCertificateInStore(s.hStore, C.MY_ENC_TYPE, 0, findType, findPara, nil); pCert != nil; pCert = C.CertFindCertificateInStore(s.hStore, C.MY_ENC_TYPE, 0, findType, findPara, pCert) {
		pCertDup := C.CertDuplicateCertificateContext(pCert)
		res = append(res, Cert{pCertDup})
	}
	return res
}

// getCert returns first of Cert's in store that satisfy findType and findPara
func (s CertStore) getCert(findType C.DWORD, findPara unsafe.Pointer) C.PCCERT_CONTEXT {
	return C.CertFindCertificateInStore(s.hStore, C.MY_ENC_TYPE, 0, findType, findPara, nil)
}

// FindBySubject returns slice of certificates with a subject that matches
// string
func (s CertStore) FindBySubject(subject string) []Cert {
	cSubject := unsafe.Pointer(C.CString(subject))
	defer C.free(cSubject)
	return s.FindCerts(C.CERT_FIND_SUBJECT_STR_A, cSubject)
}

// FindByThumb returns slice of certificates that match given thumbprint. If
// thumbprint supplied could not be decoded from string, FindByThumb will
// return nil slice
func (s CertStore) FindByThumb(thumb string) []Cert {
	bThumb, err := hex.DecodeString(thumb)
	if err != nil {
		return nil
	}
	var hashBlob C.CRYPT_HASH_BLOB
	hashBlob.cbData = C.DWORD(len(bThumb))
	bThumbPtr := C.CBytes(bThumb)
	defer C.free(bThumbPtr)
	hashBlob.pbData = (*C.BYTE)(bThumbPtr)
	return s.FindCerts(C.CERT_FIND_HASH, unsafe.Pointer(&hashBlob))
}

// GetByThumb returns first certificate in store that match given thumbprint
func (s CertStore) GetByThumb(thumb string) (res Cert, err error) {
	bThumb, err := hex.DecodeString(thumb)
	if err != nil {
		return
	}
	var hashBlob C.CRYPT_HASH_BLOB
	hashBlob.cbData = C.DWORD(len(bThumb))
	bThumbPtr := C.CBytes(bThumb)
	defer C.free(bThumbPtr)
	hashBlob.pbData = (*C.BYTE)(bThumbPtr)
	if res.pCert = s.getCert(C.CERT_FIND_HASH, unsafe.Pointer(&hashBlob)); res.pCert == nil {
		err = getErr("Error looking up certificate by thumb")
		return
	}
	return
}

// GetBySubject returns first certificate with a subject that matches
// given string
func (s CertStore) GetBySubject(subject string) (res Cert, err error) {
	cSubject := unsafe.Pointer(C.CString(subject))
	defer C.free(cSubject)

	if res.pCert = s.getCert(C.CERT_FIND_SUBJECT_STR_A, cSubject); res.pCert == nil {
		err = getErr("Error looking up certificate by subject string")
		return
	}
	return
}

// Add inserts certificate into store replacing existing certificate link if
// it's already added
func (s CertStore) Add(cert Cert) error {
	if C.CertAddCertificateContextToStore(s.hStore, cert.pCert, C.CERT_STORE_ADD_REPLACE_EXISTING, nil) == 0 {
		return getErr("Couldn't add certificate to store")
	}
	return nil
}

func (s CertStore) Certs() (res []Cert) {
	for pCert := C.CertEnumCertificatesInStore(s.hStore, nil); pCert != nil; pCert = C.CertEnumCertificatesInStore(s.hStore, pCert) {
		pCertDup := C.CertDuplicateCertificateContext(pCert)
		res = append(res, Cert{pCertDup})
	}
	return
}
