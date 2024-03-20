# Go Sessions

This package is expected to allow you to work with sessions in a Go application.

## Get started

First, creates a storage. The package provides two storage types, one is based on 
memory and the other is based on files.

    import "github.com/xandalm/go-session/memory"
    
    ...
    
    storage := memory.Storage()

or

    import "github.com/xandalm/go-session/filesystem"

    ...

    storage := filesystem.Storage()
    // or filesystem.Storage("foo/bar") to set the destination path

Now, it's necessary an adapter for the expired sessions checking step. The package 
provides an adapter based on seconds.

    adapter := session.SecondsAgeCheckerAdapter

Ok, now we can get a provider. The package has a default provider that can 
be created via NewProvider(). This function wants two args, the first is the 
storage and the second is the adapter to be used at session expiration check. 

    provider := session.NewProvider(storage, adapter)

And finally, we can get manager. The manager can be created through NewManager(). 
Pass the provider, cookie name and the expiration time unit to the function.

    manager := session.NewManager(provider, "[YOUR_COOKIE_NAME]", 3600)

  Note: The expiration time unit will be adapted through the adapter, in this case,  
  the SecondsAgeCheckerAdapter was used, which means that the expiration time is 
  3600 seconds.

Now, make a call to `manager.GC()`, which will create a routine to check for 
expired sessions, accordingly to expiration time unit.

Alright, now sessions can be create or retrieved through `manager.StartSession()`
and destroyed through `manager.DestroySession()`.

The session exposes some methods, suppose `sess` holds a session:

- `sess.SessionsID()` to get its identifier;
- `sess.Set()` to define a key and its value;
- `sess.Get()` to get a key value;
- `sess.Delete()` to remove defined key and its value.

Note: Some arguments were hidden.