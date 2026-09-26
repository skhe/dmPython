/******************************************************
file:
    exBfile.c
purpose:
    python type define for DM BFILE variables in dmPython.just be used to col description
interface:
    {}
history:
    Date        Who         RefDoc      Memo
    2017-8-17   Caichichi               Created
*******************************************************/

#include "Buffer.h"
#include "Error.h"
#include "var_pub.h"

extern DPIRETURN dpi_bfile_length(dhbfile hbfile, udint8* length);

static
void 
exBFileVar_Free(
    dm_ExternalBFile*     self    // variable to free
);

static 
PyObject*
exBFileVar_Str(
    dm_ExternalBFile* var  // variable to return the string for
);

static
PyObject*
exBFileVar_Size(
    dm_ExternalBFile* var            // variable to return the size of    
);

static 
int 
exBFileVar_InternalSize(
    dm_ExternalBFile*     var // variable to return the size of
);

static
PyObject*
exBFileVar_Read(
    dm_ExternalBFile*   var,        // variable to return the size of
    PyObject*           args,       // arguments
    PyObject*           keywordArgs // keyword arguments
);

static 
PyObject*
exBFileVar_Value(
    dm_ExternalBFile*   self,            // variable to return the size of
    int                 offset,         // offset into LOB
    slength             amount          // amount to read from LOB(IN/OUT)
);

//-----------------------------------------------------------------------------
// declaration of methods for Python type "g_BFileVarMethods"
//-----------------------------------------------------------------------------
static PyMethodDef g_ExternalBFileVarMethods[] = {
    { "size", (PyCFunction) exBFileVar_Size, METH_NOARGS },        
    { "read", (PyCFunction) exBFileVar_Read,  METH_VARARGS  | METH_KEYWORDS },
    //{ "write", (PyCFunction) exLobVar_Write, METH_VARARGS  | METH_KEYWORDS },
    //{ "truncate", (PyCFunction) exLobVar_Truncate, METH_VARARGS  | METH_KEYWORDS },        
    //{ "__reduce__", (PyCFunction) exLobVar_Reduce, METH_NOARGS },
    { NULL, NULL }
};

//-----------------------------------------------------------------------------
// Python type declaration
//-----------------------------------------------------------------------------
PyTypeObject g_exBFileVarType = {
    PyVarObject_HEAD_INIT(NULL, 0)
    "dmPython.BFILE",                   // tp_name
    sizeof(dm_ExternalBFile),           // tp_basicsize
    0,                                  // tp_itemsize
    (destructor) exBFileVar_Free,       // tp_dealloc
    0,                                  // tp_print
    0,                                  // tp_getattr
    0,                                  // tp_setattr
    0,                                  // tp_compare
    0,                                  // tp_repr
    0,                                  // tp_as_number
    0,                                  // tp_as_sequence
    0,                                  // tp_as_mapping
    0,                                  // tp_hash
    0,                                  // tp_call
    (reprfunc) exBFileVar_Str,          // tp_str
    0,                                  // tp_getattro
    0,                                  // tp_setattro
    0,                                  // tp_as_buffer
    Py_TPFLAGS_DEFAULT,                 // tp_flags
    0,                                  // tp_doc
    0,                                  // tp_traverse
    0,                                  // tp_clear
    0,                                  // tp_richcompare
    0,                                  // tp_weaklistoffset
    0,                                  // tp_iter
    0,                                  // tp_iternext
    g_ExternalBFileVarMethods,          // tp_methods
    0,                                  // tp_members
    0,                                  // tp_getset
    0,                                  // tp_base
    0,                                  // tp_dict
    0,                                  // tp_descr_get
    0,                                  // tp_descr_set
    0,                                  // tp_dictoffset
    0,                                  // tp_init
    0,                                  // tp_alloc
    0,                                  // tp_new
    0,                                  // tp_free
    0,                                  // tp_is_gc
    0                                   // tp_bases
};

//-----------------------------------------------------------------------------
// exBFileVar_Free()
//   Free an external BFile variable.
//-----------------------------------------------------------------------------
static
void
exBFileVar_Free(dm_ExternalBFile* self)
{
    dm_BFileVar* var = self->BFileVar;
    udint4 i;

    /* The bridge opens and closes the server file within each read. */
    if (var != NULL && var->data != NULL)
    {
        for (i = 0; i < var->allocatedElements; i++)
        {
            if (((dhbfile*)var->data)[i] != NULL)
                dpi_free_bfile(((dhbfile*)var->data)[i]);
            ((dhbfile*)var->data)[i] = NULL;
        }
    }

    Py_CLEAR(self->BFileVar);
    Py_TYPE(self)->tp_free((PyObject*) self);
}

// exBFileVar_Verify()
//   Verify that the external LOB var is still valid.
//-----------------------------------------------------------------------------
static
int 
exBFileVar_Verify(
    dm_ExternalBFile*      var  // variable to verify
)
{
    dm_BFileVar*           bfile_var = var->BFileVar;

    /** 连接断开，lob句柄无效；cursor关闭，bfile句柄操作可能会报错，此处增加校验 **/
    if (bfile_var->connection->isConnected <= 0)
    {
        PyErr_SetString(PyExc_ValueError, 
            "The related cursor or connection is closed");
        return -1;
    }

    return 0;
}

static 
PyObject*
exBFileVar_Str(
    dm_ExternalBFile* var  // variable to return the string for
)           
{
    PyObject*   result;
    slength     amount = 1000;

    if (exBFileVar_Verify(var) < 0)
        return NULL;

    result  = exBFileVar_Value(var, 1, amount);
    if (result == NULL)
        return NULL;

    return result;
}

//-----------------------------------------------------------------------------
// exLobVar_Size()
//   Return the size of the data in the LOB variable.
//-----------------------------------------------------------------------------
static
PyObject*
exBFileVar_Size(
    dm_ExternalBFile* var            // variable to return the size of    
)
{
    int length;

    if (exBFileVar_Verify(var) < 0)
        return NULL;

    length = exBFileVar_InternalSize(var);
    if (length < 0)
        return NULL;

#if PY_MAJOR_VERSION < 3
    return PyInt_FromLong(length);
#else
    return PyLong_FromLong(length);
#endif
}

//-----------------------------------------------------------------------------
// exBFileVar_InternalSize()
//   Return the size of the BFILE variable for internal comsumption.
//-----------------------------------------------------------------------------
static 
int
exBFileVar_InternalSize(dm_ExternalBFile* self)
{
    dm_BFileVar* var = self->BFileVar;
    udint8 length = 0;
    DPIRETURN rt;

    var->pos = self->pos;
    rt = dpi_bfile_length(((dhbfile*)var->data)[var->pos], &length);
    if (Environment_CheckForError(var->environment,
            ((dhbfile*)var->data)[var->pos], DSQL_HANDLE_BFILE, rt,
            "exBFileVar_InternalSize():dpi_bfile_length") < 0)
        return -1;

    if (length > 2147483647ULL)
    {
        PyErr_SetString(PyExc_OverflowError, "BFILE is too large for size()");
        return -1;
    }
    return (int)length;
}

// exLobVar_Value()
//   Return a portion (or all) of the data in the external LOB variable.
//-----------------------------------------------------------------------------
static
PyObject*
exBFileVar_Value(dm_ExternalBFile* self, int offset, slength amount)
{
    dm_BFileVar* var = self->BFileVar;
    DPIRETURN rt;
    sdbyte* buffer;
    udint8 data_get = 0;
    PyObject* result;

    var->pos = self->pos;
    if (amount <= 0)
        return PyBytes_FromStringAndSize("", 0);

    buffer = (sdbyte*)PyMem_Malloc((size_t)amount);
    if (buffer == NULL)
        return PyErr_NoMemory();

    rt = dpi_bfile_read(((dhbfile*)var->data)[var->pos], offset,
            DSQL_C_BINARY, amount, buffer, amount, &data_get);
    if (Environment_CheckForError(var->environment,
            ((dhbfile*)var->data)[var->pos], DSQL_HANDLE_BFILE, rt,
            "exBFileVar_Value():dpi_bfile_read") < 0)
    {
        PyMem_Free(buffer);
        return NULL;
    }

    result = PyBytes_FromStringAndSize((char*)buffer, (Py_ssize_t)data_get);
    PyMem_Free(buffer);
    return result;
}

// exBFileVar_Read()
//   Return a portion (or all) of the data in the BFILE variable.
//-----------------------------------------------------------------------------
static
PyObject*
exBFileVar_Read(
    dm_ExternalBFile*       self,        // variable to return the size of
    PyObject*               args,       // arguments
    PyObject*               keywordArgs // keyword arguments
)              
{
    dm_BFileVar*        var;
    static char         *keywordList[] = { "offset", "amount", NULL };
    int                 offset, amount;

    var         = self->BFileVar;
    var->pos    = self->pos;

    // offset and amount are expected, both optional
    offset = amount = -1;
    if (!PyArg_ParseTupleAndKeywords(args, keywordArgs, "|ii", keywordList,
        &offset, &amount))
        return NULL;

    if (offset <= 0)
    {
        offset = 1;
    }

    if (exBFileVar_Verify(self) < 0)
        return NULL;

    if (amount < 0)
    {
        amount = exBFileVar_InternalSize(self);
        if (amount < 0)
        {
            return NULL;
        }

        amount = amount - offset + 1;
        if (amount <= 0)
        {
            amount = 1;
        }
    }
    
    return exBFileVar_Value(self, offset, amount);
}

PyObject*
exBFileVar_NEW(
    dm_BFileVar*    var,        // variable to determine value for
    unsigned        pos         // array position
)
{
    dm_ExternalBFile*      self;

    self = (dm_ExternalBFile*)g_exBFileVarType.tp_alloc(&g_exBFileVarType, 0);
    if (!self)
        return NULL;

    self->pos = pos;

    Py_INCREF(var);
    self->BFileVar = var;

    return (PyObject*) self;
}
